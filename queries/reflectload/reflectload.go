// Package reflectload makes eager loading look up each model's
// Load<Relationship> method by name, for every model, instead of calling the
// model's LoadByName method. Import it for its side effect:
//
//	import _ "github.com/volatiletech/sqlboiler/queries/reflectload"
//
// A binary needs it when its models' L structs do not implement
// queries.RelationshipLoader, and can import it to stop using generated
// LoadByName methods without regenerating the models.
//
// Looking methods up by name makes the Go linker keep every exported method of
// every type stored in an interface, so a binary that imports this package is
// considerably larger.
package reflectload

import (
	"context"
	"database/sql"
	"reflect"

	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries"
)

const loadMethodPrefix = "Load"

var applicatorType = reflect.TypeOf((*queries.Applicator)(nil)).Elem()

func init() {
	queries.RegisterReflectiveLoader(load)
}

func load(loader reflect.Value, name string, ctx context.Context, exec boil.Executor, singular bool, loadingFrom reflect.Value, mods queries.Applicator) (bool, error) {
	method, found := loader.Type().MethodByName(loadMethodPrefix + name)
	if !found {
		return false, nil
	}

	args := make([]reflect.Value, 0, 6)
	args = append(args, loader)
	if ctx != nil {
		args = append(args, reflect.ValueOf(ctx))
	}
	// Hack to allow nil executors
	execArg := reflect.ValueOf(exec)
	if !execArg.IsValid() {
		execArg = reflect.ValueOf((*sql.DB)(nil))
	}
	args = append(args, execArg, reflect.ValueOf(singular), loadingFrom)
	if mods != nil {
		args = append(args, reflect.ValueOf(mods))
	} else {
		args = append(args, reflect.Zero(applicatorType))
	}

	ret := method.Func.Call(args)
	if err := ret[0].Interface(); err != nil {
		return true, err.(error)
	}
	return true, nil
}
