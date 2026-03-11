package queries

import (
	"database/sql/driver"
	"reflect"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"github.com/volatiletech/sqlboiler/drivers"
	"github.com/volatiletech/sqlboiler/queries/types"
)

var bqDialect = &drivers.Dialect{
	LQ:                '`',
	RQ:                '`',
	UseLastInsertID:   false,
	UseSchema:         false,
	UseDefaultKeyword: true,
}

var mysqlDialect = &drivers.Dialect{
	LQ:              '`',
	RQ:              '`',
	UseLastInsertID: true,
}

var pgDialect = &drivers.Dialect{
	LQ:                   '"',
	RQ:                   '"',
	UseIndexPlaceholders: true,
}

// testValuer implements driver.Valuer for testing unwrap logic.
type testValuer struct {
	val interface{}
	err error
}

func (v testValuer) Value() (driver.Value, error) {
	return v.val, v.err
}

// nestedValuer wraps another valuer to test unwrap depth limits.
// Intentionally returns a Valuer (not a valid driver.Value) to exercise
// the unwrap loop — not to simulate a real driver.
type nestedValuer struct {
	inner driver.Valuer
}

func (v nestedValuer) Value() (driver.Value, error) {
	return v.inner, nil
}

func TestTypedArgBigQuery(t *testing.T) {
	refTime := time.Date(2024, 6, 15, 10, 30, 45, 0, time.UTC)

	tests := []struct {
		name    string
		arg     TypedArgVal
		dialect *drivers.Dialect
		want    interface{}
	}{
		// BigQuery conversions
		{
			name:    "BQ DATETIME with time.Time",
			arg:     TypedArg(types.DBTypeDatetime, refTime),
			dialect: bqDialect,
			want:    civil.DateTimeOf(refTime),
		},
		{
			name:    "BQ DATE with time.Time",
			arg:     TypedArg(types.DBTypeDate, refTime),
			dialect: bqDialect,
			want:    civil.DateOf(refTime),
		},
		{
			name:    "BQ TIME with time.Time",
			arg:     TypedArg(types.DBTypeTime, refTime),
			dialect: bqDialect,
			want:    civil.TimeOf(refTime),
		},
		{
			name:    "BQ JSON with string",
			arg:     TypedArg(types.DBTypeJSON, `{"key":"val"}`),
			dialect: bqDialect,
			want:    bigquery.NullJSON{JSONVal: `{"key":"val"}`, Valid: true},
		},
		{
			name:    "BQ JSON with []byte",
			arg:     TypedArg(types.DBTypeJSON, []byte(`{"a":1}`)),
			dialect: bqDialect,
			want:    bigquery.NullJSON{JSONVal: `{"a":1}`, Valid: true},
		},
		{
			name:    "BQ GEOGRAPHY with string",
			arg:     TypedArg(types.DBTypeGeography, "POINT(1 2)"),
			dialect: bqDialect,
			want:    bigquery.NullGeography{GeographyVal: "POINT(1 2)", Valid: true},
		},

		// Nil value returns nil (let driver handle it)
		{
			name:    "BQ DATETIME nil value returns nil",
			arg:     TypedArg(types.DBTypeDatetime, nil),
			dialect: bqDialect,
			want:    nil,
		},
		{
			name:    "BQ JSON nil value returns nil",
			arg:     TypedArg(types.DBTypeJSON, nil),
			dialect: bqDialect,
			want:    nil,
		},

		// Valuer unwrap: null.T equivalent (Valuer returning nil -> bq.Null)
		{
			name:    "BQ DATETIME valuer returning nil gives NullDateTime",
			arg:     TypedArg(types.DBTypeDatetime, testValuer{val: nil, err: nil}),
			dialect: bqDialect,
			want:    bigquery.NullDateTime{},
		},
		{
			name:    "BQ JSON valuer returning nil gives NullJSON",
			arg:     TypedArg(types.DBTypeJSON, testValuer{val: nil, err: nil}),
			dialect: bqDialect,
			want:    bigquery.NullJSON{},
		},

		// Valuer unwrap: non-nil value
		{
			name:    "BQ DATETIME valuer returning time.Time",
			arg:     TypedArg(types.DBTypeDatetime, testValuer{val: refTime, err: nil}),
			dialect: bqDialect,
			want:    civil.DateTimeOf(refTime),
		},

		// Valuer unwrap: unconvertible value passes through unwrapped
		{
			name:    "BQ DATETIME valuer returning non-time value",
			arg:     TypedArg(types.DBTypeDatetime, testValuer{val: "not-a-time", err: nil}),
			dialect: bqDialect,
			want:    "not-a-time",
		},

		// Valuer unwrap: error returns original value
		{
			name:    "BQ DATETIME valuer error returns original",
			arg:     TypedArg(types.DBTypeDatetime, testValuer{val: nil, err: driver.ErrBadConn}),
			dialect: bqDialect,
			want:    testValuer{val: nil, err: driver.ErrBadConn},
		},

		// Unrecognized DBType passthrough
		{
			name:    "BQ unrecognized DBType passthrough",
			arg:     TypedArg("VARCHAR", "hello"),
			dialect: bqDialect,
			want:    "hello",
		},

		// Non-time value passthrough
		{
			name:    "BQ DATETIME with non-time value passthrough",
			arg:     TypedArg(types.DBTypeDatetime, "not-a-time"),
			dialect: bqDialect,
			want:    "not-a-time",
		},

		// UTC conversion
		{
			name:    "BQ DATETIME non-UTC converts to UTC",
			arg:     TypedArg(types.DBTypeDatetime, time.Date(2024, 6, 15, 20, 0, 0, 0, time.FixedZone("EST", -5*3600))),
			dialect: bqDialect,
			want:    civil.DateTimeOf(time.Date(2024, 6, 16, 1, 0, 0, 0, time.UTC)),
		},
		{
			name:    "BQ DATE non-UTC shifts date across boundary",
			arg:     TypedArg(types.DBTypeDate, time.Date(2024, 6, 15, 23, 0, 0, 0, time.FixedZone("EST", -5*3600))),
			dialect: bqDialect,
			want:    civil.DateOf(time.Date(2024, 6, 16, 4, 0, 0, 0, time.UTC)),
		},

		// Non-BQ dialects: passthrough
		{
			name:    "MySQL DATETIME passthrough",
			arg:     TypedArg(types.DBTypeDatetime, refTime),
			dialect: mysqlDialect,
			want:    refTime,
		},
		{
			name:    "Postgres DATETIME passthrough",
			arg:     TypedArg(types.DBTypeDatetime, refTime),
			dialect: pgDialect,
			want:    refTime,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.arg.Arg(tt.dialect)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v (%T), want %v (%T)",
					got, got, tt.want, tt.want)
			}
		})
	}
}

func TestTypedArgMaxUnwrapDepth(t *testing.T) {
	orig := maxValuerUnwrapDepth
	maxValuerUnwrapDepth = 2
	defer func() { maxValuerUnwrapDepth = orig }()

	// Create a chain deeper than maxValuerUnwrapDepth: 3 levels deep.
	innermost := testValuer{val: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	mid := nestedValuer{inner: innermost}
	outer := nestedValuer{inner: mid}

	arg := TypedArg(types.DBTypeDatetime, outer)
	got := arg.Arg(bqDialect)

	// Should return original value since unwrap depth was exceeded.
	if !reflect.DeepEqual(got, outer) {
		t.Errorf("expected original value on max depth exceeded, got %v (%T)", got, got)
	}
}

func TestResolveTypedArgsFastPath(t *testing.T) {
	plainArgs := []interface{}{"a", 42, 3.14}

	tests := []struct {
		name    string
		args    []interface{}
		dialect *drivers.Dialect
	}{
		{"BigQuery", plainArgs, bqDialect},
		{"MySQL", plainArgs, mysqlDialect},
		{"Postgres", plainArgs, pgDialect},
		{"nil dialect", plainArgs, nil},
		{"empty args", []interface{}{}, bqDialect},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveTypedArgs(tt.args, tt.dialect)
			if len(tt.args) > 0 {
				if &result[0] != &tt.args[0] {
					t.Error("expected same slice (fast path), got a copy")
				}
			}
			if !reflect.DeepEqual(result, tt.args) {
				t.Errorf("contents changed: got %v, want %v", result, tt.args)
			}
		})
	}
}

func TestResolveTypedArgsSlowPath(t *testing.T) {
	refTime := time.Date(2024, 6, 15, 10, 30, 45, 0, time.UTC)
	original := []interface{}{
		"plain",
		TypedArg(types.DBTypeDatetime, refTime),
		42,
		TypedArg(types.DBTypeDate, refTime),
		TypedArg(types.DBTypeTime, refTime),
		TypedArg(types.DBTypeJSON, `{"k":"v"}`),
		TypedArg(types.DBTypeGeography, "POINT(1 2)"),
	}

	result := resolveTypedArgs(original, bqDialect)

	expectedResult := []interface{}{
		"plain",
		civil.DateTimeOf(refTime),
		42,
		civil.DateOf(refTime),
		civil.TimeOf(refTime),
		bigquery.NullJSON{JSONVal: `{"k":"v"}`, Valid: true},
		bigquery.NullGeography{GeographyVal: "POINT(1 2)", Valid: true},
	}
	if !reflect.DeepEqual(result, expectedResult) {
		t.Errorf("resolved args mismatch:\n  got  %v\n  want %v", result, expectedResult)
	}

	expectedOrig := []interface{}{
		"plain",
		TypedArg(types.DBTypeDatetime, refTime),
		42,
		TypedArg(types.DBTypeDate, refTime),
		TypedArg(types.DBTypeTime, refTime),
		TypedArg(types.DBTypeJSON, `{"k":"v"}`),
		TypedArg(types.DBTypeGeography, "POINT(1 2)"),
	}
	if !reflect.DeepEqual(original, expectedOrig) {
		t.Error("original slice was mutated")
	}
}

func TestIsBigQueryDialect(t *testing.T) {
	tests := []struct {
		name    string
		dialect *drivers.Dialect
		want    bool
	}{
		{"BigQuery", bqDialect, true},
		{"MySQL", mysqlDialect, false},
		{"Postgres", pgDialect, false},
		{"MSSQL", &drivers.Dialect{LQ: '[', RQ: ']'}, false},
		{"SQLite", &drivers.Dialect{LQ: '"', RQ: '"'}, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBigQueryDialect(tt.dialect); got != tt.want {
				t.Errorf("isBigQueryDialect() = %v, want %v",
					got, tt.want)
			}
		})
	}
}
