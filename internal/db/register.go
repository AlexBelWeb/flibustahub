package db

import (
	"database/sql/driver"
	"fmt"
	"sync"

	"github.com/alexbelweb/flibustahub/internal/textnorm"
	"modernc.org/sqlite"
)

var registerOnce sync.Once
var registerErr error

func registerFunctions() error {
	registerOnce.Do(func() {
		registerErr = sqlite.RegisterDeterministicScalarFunction("normalize", 1, scalarString(textnorm.Normalize))
		if registerErr != nil {
			return
		}
		registerErr = sqlite.RegisterDeterministicScalarFunction("search_norm", 1, scalarString(textnorm.SearchNorm))
	})
	return registerErr
}

func scalarString(fn func(string) string) func(*sqlite.FunctionContext, []driver.Value) (driver.Value, error) {
	return func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
		if len(args) != 1 || args[0] == nil {
			return nil, nil
		}
		switch v := args[0].(type) {
		case string:
			return fn(v), nil
		case []byte:
			return fn(string(v)), nil
		default:
			return fn(fmt.Sprint(v)), nil
		}
	}
}
