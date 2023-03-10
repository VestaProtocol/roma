package stdlib

import (
	"fmt"

	"github.com/VestaProtocol/roma/x/vm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/dop251/goja"
)

const (
	Post    = "POST"
	Get     = "GET"
	Forward = "FORWARD"
)

func ApplyStandardLib(ctx sdk.Context, k Keeper, creator sdk.AccAddress, contractName string, contractAddress sdk.AccAddress, vm *goja.Runtime, readonly bool) {
	std := vm.NewObject()
	k.InitFloats(ctx, vm, std)
	initCrypto(ctx, std, vm)

	context := vm.NewObject()

	contractExports := vm.NewObject()
	contractFunctions := vm.NewObject()
	contractQueries := vm.NewObject()

	err := std.Set("panic", func(call goja.FunctionCall) goja.Value {
		message := call.Argument(0).String()
		vm.Interrupt(message)
		return goja.Undefined()
	})
	if err != nil {
		ctx.Logger().Error(err.Error())
		return
	}

	err = context.Set("sender", creator.String())
	if err != nil {
		ctx.Logger().Error(err.Error())
		return
	}
	err = contractExports.Set("address", contractAddress.String())
	if err != nil {
		ctx.Logger().Error(err.Error())
		return
	}

	err = contractExports.Set("name", contractName)
	if err != nil {
		ctx.Logger().Error(err.Error())
		return
	}

	err = contractExports.Set("functions", contractFunctions)
	if err != nil {
		ctx.Logger().Error(err.Error())
		return
	}

	err = contractExports.Set("queries", contractQueries)
	if err != nil {
		ctx.Logger().Error(err.Error())
		return
	}

	err = std.Set(Post, vm.ToValue(Post))
	if err != nil {
		ctx.Logger().Error(err.Error())
		return
	}

	err = std.Set(Forward, vm.ToValue(Forward))
	if err != nil {
		ctx.Logger().Error(err.Error())
		return
	}

	err = std.Set(Get, vm.ToValue(Get))
	if err != nil {
		ctx.Logger().Error(err.Error())
		return
	}

	for _, v := range k.Injectors() {
		module := vm.NewObject()
		err := std.Set(v.Name(), module)
		if err != nil {
			ctx.Logger().Error(err.Error())
			return
		}

		err = v.Inject(ctx, vm, creator, contractName, contractAddress, module, readonly)
		if err != nil {
			ctx.Logger().Error(err.Error())
			return
		}
	}

	if !readonly {
		err = std.Set("write", func(call goja.FunctionCall) goja.Value {
			key := call.Argument(0).String()
			data := call.Argument(1).String()

			size, ok := sdk.NewIntFromString(fmt.Sprintf("%d", len(data)))
			if !ok {
				return goja.ValueFalse()
			}

			toSave := types.Romdata{
				Index: fmt.Sprintf("%s%s", contractAddress.String(), key),
				Data:  data,
			}

			k.SetRomdata(ctx, toSave)

			ctx.GasMeter().ConsumeGas(size.Uint64()*types.DefaultGasValues().Write, "saving data")

			return goja.ValueTrue()
		})
		if err != nil {
			ctx.Logger().Error(err.Error())
			return
		}

		err = std.Set("delete", func(call goja.FunctionCall) goja.Value {
			key := call.Argument(0).String()

			k.RemoveRomdata(ctx, fmt.Sprintf("%s%s", contractAddress.String(), key))

			ctx.GasMeter().ConsumeGas(types.DefaultGasValues().Write, "deleting data")

			return goja.ValueTrue()
		})
		if err != nil {
			ctx.Logger().Error(err.Error())
			return
		}
	}

	err = std.Set("read", func(call goja.FunctionCall) goja.Value {
		key := call.Argument(0).String()

		data, found := k.GetRomdata(ctx, fmt.Sprintf("%s%s", contractAddress.String(), key))
		if !found {
			ctx.Logger().Info(fmt.Sprintf("Cannot find %s", key))
			return goja.Null()
		}

		l := len(data.Data)

		size, ok := sdk.NewIntFromString(fmt.Sprintf("%d", l))
		if !ok {
			ctx.Logger().Info(fmt.Sprintf("Cannot parse %d", l))
			return goja.Null()
		}

		ctx.GasMeter().ConsumeGas(size.Uint64()*types.DefaultGasValues().Read, "reading data")

		val := vm.ToValue(data.Data)

		return val
	})
	if err != nil {
		ctx.Logger().Error(err.Error())
		return
	}

	err = std.Set("require", func(call goja.FunctionCall) goja.Value {
		moduleName := call.Argument(0).String()
		versionValue := call.Argument(1).String()

		ctx.GasMeter().ConsumeGas(types.DefaultGasValues().Import, "importing library")

		program, found := k.GetProgram(ctx, moduleName)
		if !found {
			return goja.Null()
		}

		v := k.GetContractVersion(program, versionValue)
		ctx.Logger().Debug(fmt.Sprintf("Requested version: %d", v))

		source, ok := k.GetContracts(ctx, v)
		if !ok {
			return goja.Null()
		}

		code := source.Source

		newvm := goja.New()
		ApplyStandardLib(ctx, k, creator, contractName, contractAddress, newvm, readonly)

		_, err := newvm.RunString(code)
		if err != nil {
			return goja.Null()
		}

		ctx.Logger().Info("completed creation of temp vm")

		ctc := newvm.Get("CONTRACT").ToObject(newvm)
		if ctc == nil {
			e := sdkerrors.Wrapf(sdkerrors.ErrKeyNotFound, "cannot find contract")
			ctx.Logger().Error(e.Error())
			return goja.Null()
		}

		ctx.Logger().Info("returning object")

		return ctc
	})
	if err != nil {
		ctx.Logger().Error(err.Error())
		return
	}

	err = std.Set("fetch", func(call goja.FunctionCall) goja.Value {
		moduleName := call.Argument(0).String()
		entryPoint := call.Argument(1).String()
		fetchType := call.Argument(2).String()
		args := call.Arguments[3:]

		ctx.GasMeter().ConsumeGas(types.DefaultGasValues().Import, "fetching contract")

		program, found := k.GetProgram(ctx, moduleName)
		if !found {
			return throwError(fmt.Sprintf("Cannot find contract %s", moduleName), vm)
		}
		v := k.GetContractVersion(program, "-1")
		source, ok := k.GetContracts(ctx, v)
		if !ok {
			return throwError(fmt.Sprintf("Cannot find program %d", v), vm)
		}

		code := source.Source

		var res string
		var err error

		if fetchType == Post {
			res, err = k.ExecuteContract(ctx, program.Name, code, entryPoint, contractAddress, args)
			if err != nil {
				return throwError(err.Error(), vm)
			}
		} else if fetchType == Forward {
			res, err = k.ExecuteContract(ctx, program.Name, code, entryPoint, creator, args)
			if err != nil {
				return throwError(err.Error(), vm)
			}
		} else if fetchType == Get {
			res, err = k.QueryContract(ctx, program.Name, code, entryPoint, args)
			if err != nil {
				return throwError(err.Error(), vm)
			}
		} else {
			return throwError("no feetch method specified", vm)
		}

		return vm.ToValue(res)
	})
	if err != nil {
		ctx.Logger().Error(err.Error())
		return
	}

	err = vm.Set("STD", std)
	if err != nil {
		ctx.Logger().Error(err.Error())
		return
	}

	err = vm.Set("CTX", context)
	if err != nil {
		ctx.Logger().Error(err.Error())
		return
	}

	err = vm.Set("CONTRACT", contractExports)
	if err != nil {
		ctx.Logger().Error(err.Error())
		return
	}
}
