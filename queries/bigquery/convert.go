package bigquery

import (
	"time"

	bq "cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
)

// ConvertArg converts a value to the BigQuery-appropriate type based on
// the database column type string. Returns the value unchanged for
// unrecognized types.
func ConvertArg(dbType string, value interface{}) interface{} {
	switch dbType {
	case "DATETIME":
		if value == nil {
			return bq.NullDateTime{}
		}
		if tm, ok := value.(time.Time); ok {
			return civil.DateTimeOf(tm.UTC())
		}
	case "DATE":
		if value == nil {
			return bq.NullDate{}
		}
		if tm, ok := value.(time.Time); ok {
			return civil.DateOf(tm.UTC())
		}
	case "TIME":
		if value == nil {
			return bq.NullTime{}
		}
		if tm, ok := value.(time.Time); ok {
			return civil.TimeOf(tm.UTC())
		}
	case "JSON":
		if value == nil {
			return bq.NullJSON{}
		}
		switch s := value.(type) {
		case string:
			return bq.NullJSON{JSONVal: s, Valid: true}
		case []byte:
			return bq.NullJSON{JSONVal: string(s), Valid: true}
		}
	case "GEOGRAPHY":
		if value == nil {
			return bq.NullGeography{}
		}
		if s, ok := value.(string); ok {
			return bq.NullGeography{GeographyVal: s, Valid: true}
		}
	}
	return value
}
