package bigquery

import (
	"reflect"
	"testing"
	"time"

	bq "cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
)

func TestConvertArg(t *testing.T) {
	refTime := time.Date(2024, 6, 15, 10, 30, 45, 0, time.UTC)

	tests := []struct {
		name   string
		dbType string
		value  interface{}
		want   interface{}
	}{
		{
			name:   "DATETIME with time.Time",
			dbType: "DATETIME",
			value:  refTime,
			want:   civil.DateTimeOf(refTime),
		},
		{
			name:   "DATE with time.Time",
			dbType: "DATE",
			value:  refTime,
			want:   civil.DateOf(refTime),
		},
		{
			name:   "TIME with time.Time",
			dbType: "TIME",
			value:  refTime,
			want:   civil.TimeOf(refTime),
		},
		{
			name:   "JSON with string",
			dbType: "JSON",
			value:  `{"key":"val"}`,
			want:   bq.NullJSON{JSONVal: `{"key":"val"}`, Valid: true},
		},
		{
			name:   "JSON with []byte",
			dbType: "JSON",
			value:  []byte(`{"a":1}`),
			want:   bq.NullJSON{JSONVal: `{"a":1}`, Valid: true},
		},
		{
			name:   "GEOGRAPHY with string",
			dbType: "GEOGRAPHY",
			value:  "POINT(1 2)",
			want:   bq.NullGeography{GeographyVal: "POINT(1 2)", Valid: true},
		},
		{
			name:   "DATETIME nil value",
			dbType: "DATETIME",
			value:  nil,
			want:   bq.NullDateTime{},
		},
		{
			name:   "DATE nil value",
			dbType: "DATE",
			value:  nil,
			want:   bq.NullDate{},
		},
		{
			name:   "TIME nil value",
			dbType: "TIME",
			value:  nil,
			want:   bq.NullTime{},
		},
		{
			name:   "JSON nil value",
			dbType: "JSON",
			value:  nil,
			want:   bq.NullJSON{},
		},
		{
			name:   "GEOGRAPHY nil value",
			dbType: "GEOGRAPHY",
			value:  nil,
			want:   bq.NullGeography{},
		},
		{
			name:   "unrecognized DBType passthrough",
			dbType: "VARCHAR",
			value:  "hello",
			want:   "hello",
		},
		{
			name:   "DATETIME with non-UTC time converts to UTC",
			dbType: "DATETIME",
			value:  time.Date(2024, 6, 15, 20, 0, 0, 0, time.FixedZone("EST", -5*3600)),
			want:   civil.DateTimeOf(time.Date(2024, 6, 16, 1, 0, 0, 0, time.UTC)),
		},
		{
			name:   "DATE with non-UTC time shifts date across boundary",
			dbType: "DATE",
			value:  time.Date(2024, 6, 15, 23, 0, 0, 0, time.FixedZone("EST", -5*3600)),
			want:   civil.DateOf(time.Date(2024, 6, 16, 4, 0, 0, 0, time.UTC)),
		},
		{
			name:   "DATETIME with non-time value passthrough",
			dbType: "DATETIME",
			value:  "not-a-time",
			want:   "not-a-time",
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
