package queries

import (
	"context"
	"reflect"

	"github.com/volatiletech/sqlboiler/boil"
)

// RelationshipLoader is implemented by a generated model's L struct.
// LoadByName runs the eager load for the relationship called name, the same
// work as calling the struct's Load<name> method directly, and reports
// found=false when name is not one of the model's relationships. exec is the
// executor passed to Bind; each model converts it to the executor type its
// Load methods take.
//
// Eager loading dispatches through this interface so that binaries never
// look methods up by name at runtime, which would make the Go linker keep
// every exported method of every type stored in an interface.
type RelationshipLoader interface {
	LoadByName(name string, ctx context.Context, exec interface{}, singular bool, maybe interface{}, mods Applicator) (found bool, err error)
}

// ReflectiveLoadFunc eager loads the relationship called name by looking up
// the Load<name> method on loader through reflection. It returns found=false
// when loader has no such method.
type ReflectiveLoadFunc func(loader reflect.Value, name string, ctx context.Context, exec boil.Executor, singular bool, loadingFrom reflect.Value, mods Applicator) (found bool, err error)

var reflectiveLoad ReflectiveLoadFunc

// RegisterReflectiveLoader installs a reflective loader that eager loading
// uses for every model, in place of LoadByName. It makes models whose L struct
// does not implement RelationshipLoader loadable, and lets a binary go back to
// the reflective lookup without regenerating its models. Package
// queries/reflectload registers it on import; binaries that do not import that
// package never link the reflective lookup.
func RegisterReflectiveLoader(fn ReflectiveLoadFunc) {
	reflectiveLoad = fn
}
