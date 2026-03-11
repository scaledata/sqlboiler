package bigquery

import (
	"reflect"
	"testing"
	"time"

	bq "cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"github.com/volatiletech/sqlboiler/queries/types"
)

func TestConvertArg(t *testing.T) {
	refTime := time.Date(2024, 6, 15, 10, 30, 45, 0, time.UTC)

	tests := []struct {
		name   string
		dbType types.DBType
		value  interface{}
		want   interface{}
	}{
		// DATETIME conversions
		{
			name:   "DATETIME with time.Time",
			dbType: types.DBTypeDatetime,
			value:  refTime,
			want:   civil.DateTimeOf(refTime),
		},
		{
			name:   "DATETIME with non-UTC time converts to UTC",
			dbType: types.DBTypeDatetime,
			value:  time.Date(2024, 6, 15, 20, 0, 0, 0, time.FixedZone("EST", -5*3600)),
			want:   civil.DateTimeOf(time.Date(2024, 6, 16, 1, 0, 0, 0, time.UTC)),
		},
		{
			name:   "DATETIME with non-time value passthrough",
			dbType: types.DBTypeDatetime,
			value:  "not-a-time",
			want:   "not-a-time",
		},
		{
			name:   "DATETIME nil returns NullDateTime",
			dbType: types.DBTypeDatetime,
			value:  nil,
			want:   bq.NullDateTime{},
		},

		// DATE conversions
		{
			name:   "DATE with time.Time",
			dbType: types.DBTypeDate,
			value:  refTime,
			want:   civil.DateOf(refTime),
		},
		{
			name:   "DATE with non-UTC time shifts date across boundary",
			dbType: types.DBTypeDate,
			value:  time.Date(2024, 6, 15, 23, 0, 0, 0, time.FixedZone("EST", -5*3600)),
			want:   civil.DateOf(time.Date(2024, 6, 16, 4, 0, 0, 0, time.UTC)),
		},
		{
			name:   "DATE nil returns NullDate",
			dbType: types.DBTypeDate,
			value:  nil,
			want:   bq.NullDate{},
		},

		// TIME conversions
		{
			name:   "TIME with time.Time",
			dbType: types.DBTypeTime,
			value:  refTime,
			want:   civil.TimeOf(refTime),
		},
		{
			name:   "TIME nil returns NullTime",
			dbType: types.DBTypeTime,
			value:  nil,
			want:   bq.NullTime{},
		},

		// JSON conversions
		{
			name:   "JSON with string",
			dbType: types.DBTypeJSON,
			value:  `{"key":"val"}`,
			want:   bq.NullJSON{JSONVal: `{"key":"val"}`, Valid: true},
		},
		{
			name:   "JSON with []byte",
			dbType: types.DBTypeJSON,
			value:  []byte(`{"a":1}`),
			want:   bq.NullJSON{JSONVal: `{"a":1}`, Valid: true},
		},
		{
			name:   "JSON nil returns NullJSON",
			dbType: types.DBTypeJSON,
			value:  nil,
			want:   bq.NullJSON{},
		},

		// GEOGRAPHY conversions
		{
			name:   "GEOGRAPHY with string",
			dbType: types.DBTypeGeography,
			value:  "POINT(1 2)",
			want:   bq.NullGeography{GeographyVal: "POINT(1 2)", Valid: true},
		},
		{
			name:   "GEOGRAPHY nil returns NullGeography",
			dbType: types.DBTypeGeography,
			value:  nil,
			want:   bq.NullGeography{},
		},

		// Unrecognized type
		{
			name:   "unrecognized DBType passthrough",
			dbType: "VARCHAR",
			value:  "hello",
			want:   "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertArg(tt.dbType, tt.value)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertArg(%q, %v) = %v (%T), want %v (%T)",
					tt.dbType, tt.value, got, got, tt.want, tt.want)
			}
		})
	}
}
