package queries

import (
	"database/sql/driver"

	bqconv "github.com/volatiletech/sqlboiler/queries/bigquery"
	"github.com/volatiletech/sqlboiler/queries/types"
	"github.com/volatiletech/sqlboiler/drivers"
)

// maxValuerUnwrapDepth is the maximum number of driver.Valuer unwrap
// iterations, matching database/sql behavior. Exposed as a var so
// tests can override it.
var maxValuerUnwrapDepth = 10

// TypedArgVal wraps a query argument with its database column type,
// enabling dialect-specific type conversion at query build time.
type TypedArgVal struct {
	dbType types.DBType
	value  interface{}
}

// TypedArg creates a TypedArgVal with the given database type and value.
func TypedArg(dbType types.DBType, value interface{}) TypedArgVal {
	return TypedArgVal{dbType: dbType, value: value}
}

// Arg converts the value to the dialect-appropriate type.
func (t TypedArgVal) Arg(dialect *drivers.Dialect) interface{} {
	if !isBigQueryDialect(dialect) || t.value == nil {
		return t.value
	}

	val := t.value

	// Unwrap driver.Valuer (max iterations matching database/sql).
	for i := 0; i < maxValuerUnwrapDepth; i++ {
		v, ok := val.(driver.Valuer)
		if !ok {
			break
		}
		var err error
		val, err = v.Value()
		if err != nil {
			return t.value
		}
	}

	// If we exhausted iterations and val is still a Valuer, return
	// the original value and let the driver handle it.
	if _, ok := val.(driver.Valuer); ok {
		return t.value
	}

	return bqconv.ConvertArg(t.dbType, val)
}

// isBigQueryDialect detects BigQuery by its dialect characteristics:
// backtick quoting and no LastInsertID support.
//
// TODO: Make platform a native field of queries.Query (set alongside
// dialect in OLAP templates) so we can use platform == BigQuery instead
// of relying on dialect field heuristics, which may collide if another
// platform shares the same dialect characteristics.
func isBigQueryDialect(d *drivers.Dialect) bool {
	return d != nil && d.LQ == '`' && !d.UseLastInsertID
}

// resolveTypedArgs resolves any TypedArgVal values in the args slice
// using the given dialect. Returns the original slice unchanged if
// no TypedArgVal values are present (zero allocation fast path).
func resolveTypedArgs(args []interface{}, dialect *drivers.Dialect) []interface{} {
	if dialect == nil {
		return args
	}

	hasTypedArg := false
	for _, a := range args {
		if _, ok := a.(TypedArgVal); ok {
			hasTypedArg = true
			break
		}
	}
	if !hasTypedArg {
		return args
	}

	resolved := make([]interface{}, len(args))
	for i, a := range args {
		if ta, ok := a.(TypedArgVal); ok {
			resolved[i] = ta.Arg(dialect)
		} else {
			resolved[i] = a
		}
	}
	return resolved
}
