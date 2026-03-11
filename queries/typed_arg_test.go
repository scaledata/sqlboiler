package queries

import (
	"database/sql/driver"
	"reflect"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"github.com/volatiletech/sqlboiler/drivers"
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

func TestTypedArgBigQuery(t *testing.T) {
	refTime := time.Date(2024, 6, 15, 10, 30, 45, 0, time.UTC)

	tests := []struct {
		name string
		arg  TypedArg
		want interface{}
	}{
		{
			name: "DATETIME with time.Time",
			arg:  TypedArg{DBType: "DATETIME", Value: refTime},
			want: civil.DateTimeOf(refTime),
		},
		{
			name: "DATE with time.Time",
			arg:  TypedArg{DBType: "DATE", Value: refTime},
			want: civil.DateOf(refTime),
		},
		{
			name: "TIME with time.Time",
			arg:  TypedArg{DBType: "TIME", Value: refTime},
			want: civil.TimeOf(refTime),
		},
		{
			name: "JSON with string",
			arg:  TypedArg{DBType: "JSON", Value: `{"key":"val"}`},
			want: bigquery.NullJSON{JSONVal: `{"key":"val"}`, Valid: true},
		},
		{
			name: "JSON with []byte",
			arg:  TypedArg{DBType: "JSON", Value: []byte(`{"a":1}`)},
			want: bigquery.NullJSON{JSONVal: `{"a":1}`, Valid: true},
		},
		{
			name: "GEOGRAPHY with string",
			arg:  TypedArg{DBType: "GEOGRAPHY", Value: "POINT(1 2)"},
			want: bigquery.NullGeography{GeographyVal: "POINT(1 2)", Valid: true},
		},
		{
			name: "DATETIME nil value",
			arg:  TypedArg{DBType: "DATETIME", Value: nil},
			want: bigquery.NullDateTime{},
		},
		{
			name: "DATE nil value",
			arg:  TypedArg{DBType: "DATE", Value: nil},
			want: bigquery.NullDate{},
		},
		{
			name: "TIME nil value",
			arg:  TypedArg{DBType: "TIME", Value: nil},
			want: bigquery.NullTime{},
		},
		{
			name: "JSON nil value",
			arg:  TypedArg{DBType: "JSON", Value: nil},
			want: bigquery.NullJSON{},
		},
		{
			name: "GEOGRAPHY nil value",
			arg:  TypedArg{DBType: "GEOGRAPHY", Value: nil},
			want: bigquery.NullGeography{},
		},
		{
			name: "unrecognized DBType passthrough",
			arg:  TypedArg{DBType: "VARCHAR", Value: "hello"},
			want: "hello",
		},
		{
			name: "DATETIME with non-UTC time converts to UTC",
			arg:  TypedArg{DBType: "DATETIME", Value: time.Date(2024, 6, 15, 20, 0, 0, 0, time.FixedZone("EST", -5*3600))},
			want: civil.DateTimeOf(time.Date(2024, 6, 16, 1, 0, 0, 0, time.UTC)),
		},
		{
			name: "DATE with non-UTC time shifts date across boundary",
			arg:  TypedArg{DBType: "DATE", Value: time.Date(2024, 6, 15, 23, 0, 0, 0, time.FixedZone("EST", -5*3600))},
			want: civil.DateOf(time.Date(2024, 6, 16, 4, 0, 0, 0, time.UTC)),
		},
		{
			name: "DATETIME with non-time value passthrough",
			arg:  TypedArg{DBType: "DATETIME", Value: "not-a-time"},
			want: "not-a-time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.arg.Arg(bqDialect)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v (%T), want %v (%T)",
					got, got, tt.want, tt.want)
			}
		})
	}
}

func TestTypedArgNonBQDialect(t *testing.T) {
	refTime := time.Date(2024, 6, 15, 10, 30, 45, 0, time.UTC)

	tests := []struct {
		name    string
		dialect *drivers.Dialect
	}{
		{"MySQL", mysqlDialect},
		{"Postgres", pgDialect},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			arg := TypedArg{DBType: "DATETIME", Value: refTime}
			got := arg.Arg(tt.dialect)
			if got != refTime {
				t.Errorf("expected passthrough for %s, got %v", tt.name, got)
			}
		})
	}
}

func TestTypedArgValuerUnwrap(t *testing.T) {
	refTime := time.Date(2024, 6, 15, 10, 30, 45, 0, time.UTC)

	t.Run("valuer returning time.Time", func(t *testing.T) {
		arg := TypedArg{
			DBType: "DATETIME",
			Value:  testValuer{val: refTime, err: nil},
		}
		got := arg.Arg(bqDialect)
		want := civil.DateTimeOf(refTime)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("valuer returning nil", func(t *testing.T) {
		arg := TypedArg{
			DBType: "JSON",
			Value:  testValuer{val: nil, err: nil},
		}
		got := arg.Arg(bqDialect)
		want := bigquery.NullJSON{}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("valuer returning error", func(t *testing.T) {
		original := testValuer{
			val: nil,
			err: driver.ErrBadConn,
		}
		arg := TypedArg{
			DBType: "DATETIME",
			Value:  original,
		}
		got := arg.Arg(bqDialect)
		// On error, original value is returned unchanged.
		if got != original {
			t.Errorf("expected original value on error, got %v", got)
		}
	})
}

func TestResolveTypedArgsFastPath(t *testing.T) {
	args := []interface{}{"a", 42, 3.14}
	result := resolveTypedArgs(args, bqDialect)
	// Fast path: same slice returned when no TypedArg present.
	if &result[0] != &args[0] {
		t.Error("expected same slice (fast path), got a copy")
	}
}

func TestResolveTypedArgsSlowPath(t *testing.T) {
	refTime := time.Date(2024, 6, 15, 10, 30, 45, 0, time.UTC)
	original := []interface{}{
		"plain",
		TypedArg{DBType: "DATETIME", Value: refTime},
		42,
	}
	result := resolveTypedArgs(original, bqDialect)

	// Slow path: new slice.
	if &result[0] == &original[0] {
		t.Error("expected new slice (slow path), got same")
	}

	// Non-TypedArg values unchanged.
	if result[0] != "plain" {
		t.Errorf("expected 'plain', got %v", result[0])
	}
	if result[2] != 42 {
		t.Errorf("expected 42, got %v", result[2])
	}

	// TypedArg resolved.
	want := civil.DateTimeOf(refTime)
	if !reflect.DeepEqual(result[1], want) {
		t.Errorf("got %v, want %v", result[1], want)
	}

	// Original slice unchanged (still holds TypedArg).
	if _, ok := original[1].(TypedArg); !ok {
		t.Error("original slice was mutated")
	}
}

func TestResolveTypedArgsNilDialect(t *testing.T) {
	args := []interface{}{
		TypedArg{DBType: "DATETIME", Value: time.Now()},
	}
	result := resolveTypedArgs(args, nil)
	// Nil dialect: return unchanged. In production, dialect is always set
	// by generated ORM code via SetDialect. This guard prevents a panic
	// if dialect is somehow nil; the raw TypedArg will reach the driver
	// and fail at exec time with a descriptive error.
	if &result[0] != &args[0] {
		t.Error("expected same slice with nil dialect")
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
