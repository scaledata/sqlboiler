package queries

import (
	"database/sql/driver"

	bqconv "github.com/volatiletech/sqlboiler/queries/bigquery"
	"github.com/volatiletech/sqlboiler/drivers"
)

// DBType identifies a database column type for dialect-specific conversion.
type DBType string

const (
	DBTypeDatetime  DBType = "DATETIME"
	DBTypeDate      DBType = "DATE"
	DBTypeTime      DBType = "TIME"
	DBTypeJSON      DBType = "JSON"
	DBTypeGeography DBType = "GEOGRAPHY"
)

// TypedArg wraps a query argument with its database column type,
// enabling dialect-specific type conversion at query build time.
type TypedArg struct {
	dbType DBType
	value  interface{}
}

// NewTypedArg creates a TypedArg with the given database type and value.
func NewTypedArg(dbType DBType, value interface{}) TypedArg {
	return TypedArg{dbType: dbType, value: value}
}

// Arg converts the value to the dialect-appropriate type.
func (t TypedArg) Arg(dialect *drivers.Dialect) interface{} {
	if !isBigQueryDialect(dialect) {
		return t.value
	}

	val := t.value

	// Unwrap driver.Valuer (max 10 iterations, matching database/sql).
	for i := 0; i < 10; i++ {
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

	result := bqconv.ConvertArg(string(t.dbType), val)
	if result != val {
		return result
	}
	return t.value
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

// resolveTypedArgs resolves any TypedArg values in the args slice
// using the given dialect. Returns the original slice unchanged if
// no TypedArg values are present (zero allocation fast path).
func resolveTypedArgs(args []interface{}, dialect *drivers.Dialect) []interface{} {
	if dialect == nil {
		return args
	}

	hasTypedArg := false
	for _, a := range args {
		if _, ok := a.(TypedArg); ok {
			hasTypedArg = true
			break
		}
	}
	if !hasTypedArg {
		return args
	}

	resolved := make([]interface{}, len(args))
	for i, a := range args {
		if ta, ok := a.(TypedArg); ok {
			resolved[i] = ta.Arg(dialect)
		} else {
			resolved[i] = a
		}
	}
	return resolved
}
