package queries

import (
	"database/sql/driver"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"github.com/volatiletech/sqlboiler/drivers"
)

// TypedArg wraps a query argument with its database column type,
// enabling dialect-specific type conversion at query build time.
type TypedArg struct {
	DBType string
	Value  interface{}
}

// Arg converts the value to the dialect-appropriate type.
func (t TypedArg) Arg(dialect *drivers.Dialect) interface{} {
	if !isBigQueryDialect(dialect) {
		return t.Value
	}

	val := t.Value

	// Unwrap driver.Valuer (max 10 iterations, matching database/sql).
	for i := 0; i < 10; i++ {
		v, ok := val.(driver.Valuer)
		if !ok {
			break
		}
		var err error
		val, err = v.Value()
		if err != nil {
			return t.Value
		}
	}

	if val == nil {
		return nullForDBType(t.DBType)
	}

	switch t.DBType {
	case "DATETIME":
		if tm, ok := val.(time.Time); ok {
			return civil.DateTimeOf(tm.UTC())
		}
	case "DATE":
		if tm, ok := val.(time.Time); ok {
			return civil.DateOf(tm.UTC())
		}
	case "TIME":
		if tm, ok := val.(time.Time); ok {
			return civil.TimeOf(tm.UTC())
		}
	case "JSON":
		switch s := val.(type) {
		case string:
			return bigquery.NullJSON{JSONVal: s, Valid: true}
		case []byte:
			return bigquery.NullJSON{JSONVal: string(s), Valid: true}
		}
	case "GEOGRAPHY":
		if s, ok := val.(string); ok {
			return bigquery.NullGeography{GeographyVal: s, Valid: true}
		}
	}

	return val
}

func nullForDBType(dbType string) interface{} {
	switch dbType {
	case "DATETIME":
		return bigquery.NullDateTime{}
	case "DATE":
		return bigquery.NullDate{}
	case "TIME":
		return bigquery.NullTime{}
	case "JSON":
		return bigquery.NullJSON{}
	case "GEOGRAPHY":
		return bigquery.NullGeography{}
	default:
		return nil
	}
}

// isBigQueryDialect detects BigQuery by its dialect characteristics:
// backtick quoting and no LastInsertID support.
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

	for _, a := range args {
		if _, ok := a.(TypedArg); ok {
			goto resolve
		}
	}
	return args

resolve:
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
