package stdlib

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/VestaProtocol/roma/x/vm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dop251/goja"
)

func initCrypto(ctx sdk.Context, std *goja.Object, vm *goja.Runtime) {
	crypto := vm.NewObject()

	// sha256 declaration
	err := crypto.Set("sha256", func(call goja.FunctionCall) goja.Value {
		data := call.Argument(0).String()

		h := sha256.New()
		h.Write([]byte(data))
		res := h.Sum(nil)
		hexData := hex.EncodeToString(res)

		ctx.GasMeter().ConsumeGas(types.DefaultGasValues().Hash, "hashing data")

		val := vm.ToValue(hexData)

		return val
	})
	if err != nil {
		ctx.Logger().Error(err.Error())
		return
	}

	err = std.Set("crypto", crypto)
	if err != nil {
		ctx.Logger().Error(err.Error())
		return
	}
}
