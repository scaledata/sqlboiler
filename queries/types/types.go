package types

// DBType identifies a database column type for dialect-specific conversion.
type DBType string

const (
	DBTypeDatetime  DBType = "DATETIME"
	DBTypeDate      DBType = "DATE"
	DBTypeTime      DBType = "TIME"
	DBTypeJSON      DBType = "JSON"
	DBTypeGeography DBType = "GEOGRAPHY"
)
