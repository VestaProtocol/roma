package stdlib

import (
	"github.com/dop251/goja"
)

func throwError(message string, vm *goja.Runtime) *goja.Object {
	er := vm.NewObject()
	_ = er.Set("message", message)

	return er
}
