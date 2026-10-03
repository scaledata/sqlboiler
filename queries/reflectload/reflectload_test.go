package reflectload

import (
	"reflect"
	"testing"

	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries"
)

type reflectOnlyModel struct {
	loaded bool
}

type reflectOnlyL struct{}

func (reflectOnlyL) LoadThing(_ boil.Executor, singular bool, maybe interface{}, _ queries.Applicator) error {
	if singular {
		maybe.(*reflectOnlyModel).loaded = true
	}
	return nil
}

func TestLoadFindsMethodByName(t *testing.T) {
	obj := &reflectOnlyModel{}
	found, err := load(reflect.Zero(reflect.TypeOf(reflectOnlyL{})), "Thing", nil, nil, true, reflect.ValueOf(obj), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected LoadThing to be found")
	}
	if !obj.loaded {
		t.Error("expected LoadThing to run")
	}
}

func TestLoadReportsMissingMethod(t *testing.T) {
	found, err := load(reflect.Zero(reflect.TypeOf(reflectOnlyL{})), "Other", nil, nil, true, reflect.ValueOf(&reflectOnlyModel{}), nil)
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Error("expected LoadOther not to be found")
	}
}
