package keeper

import (
	"github.com/VestaProtocol/roma/x/vm/keeper/stdlib"
	"github.com/VestaProtocol/roma/x/vm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/dop251/goja"
	"github.com/tendermint/tendermint/crypto"
)

func (k Keeper) GetContractVersion(program types.Program, version string) uint64 {
	l := len(program.Code) - 1

	v, ok := sdk.NewIntFromString(version)
	if !ok {
		return program.Code[l]
	}

	if v.Int64() < 0 {
		return program.Code[l]
	}

	return program.Code[v.Int64()]
}

func NewContractAddress(name string) sdk.AccAddress {
	return sdk.AccAddress(crypto.AddressHash([]byte(name)))
}

func (k Keeper) newContractAccount(ctx sdk.Context, address sdk.AccAddress) authtypes.AccountI {
	baseAcc := k.accountKeeper.NewAccountWithAddress(ctx, address)

	return baseAcc
}

func (k Keeper) getContractAddress(ctx sdk.Context, contractName string) (sdk.AccAddress, error) {
	address := NewContractAddress(contractName)

	acc := k.accountKeeper.GetAccount(ctx, address)
	if acc != nil {
		return acc.GetAddress(), nil
	}

	acc = k.newContractAccount(ctx, address)

	k.accountKeeper.SetAccount(ctx, acc)

	return acc.GetAddress(), nil
}

func (k Keeper) InitContract(ctx sdk.Context, name string, sourceCode string, creator sdk.AccAddress, args []goja.Value) (string, error) {
	vm, err := k.buildContract(ctx, name, sourceCode, creator, false)
	if err != nil {
		return "", err
	}

	ctc := vm.Get("CONTRACT").ToObject(vm)
	if ctc == nil {
		e := sdkerrors.Wrapf(sdkerrors.ErrKeyNotFound, "cannot find contract")
		ctx.Logger().Error(e.Error())
		return "", e
	}

	f := ctc.Get("init")
	if f == nil {
		e := sdkerrors.Wrapf(sdkerrors.ErrKeyNotFound, "cannot find entry %s", "init")
		ctx.Logger().Error(e.Error())
		return "", nil
	}

	function, ok := goja.AssertFunction(f)
	if !ok {
		e := sdkerrors.Wrapf(sdkerrors.ErrKeyNotFound, "cannot get %s from CONTRACT", "init")
		ctx.Logger().Error(e.Error())
		return "", e
	}

	res, err := function(goja.Undefined(), args...)
	if err != nil {
		e := sdkerrors.Wrapf(err, "cannot run function")
		ctx.Logger().Error(e.Error())
		return "", e
	}

	return res.String(), nil
}

func (k Keeper) ExecuteContract(ctx sdk.Context, name string, sourceCode string, entry string, creator sdk.AccAddress, args []goja.Value) (string, error) {
	vm, err := k.buildContract(ctx, name, sourceCode, creator, false)
	if err != nil {
		return "", err
	}

	ctc := vm.Get("CONTRACT").ToObject(vm)
	if ctc == nil {
		e := sdkerrors.Wrapf(sdkerrors.ErrKeyNotFound, "cannot find contract")
		ctx.Logger().Error(e.Error())
		return "", e
	}

	r := ctc.Get("functions").ToObject(vm)
	if r == nil {
		e := sdkerrors.Wrapf(sdkerrors.ErrKeyNotFound, "cannot find functions")
		ctx.Logger().Error(e.Error())
		return "", e
	}

	f := r.Get(entry)
	if f == nil {
		e := sdkerrors.Wrapf(sdkerrors.ErrKeyNotFound, "cannot find entry %s", entry)
		ctx.Logger().Error(e.Error())
		return "", e
	}

	function, ok := goja.AssertFunction(f)
	if !ok {
		e := sdkerrors.Wrapf(sdkerrors.ErrKeyNotFound, "cannot get %s from contractFunctions", entry)
		ctx.Logger().Error(e.Error())
		return "", e
	}

	res, err := function(goja.Undefined(), args...)
	if err != nil {
		e := sdkerrors.Wrapf(err, "cannot run function")
		ctx.Logger().Error(e.Error())
		return "", e
	}

	return res.String(), nil
}

func (k Keeper) QueryContract(ctx sdk.Context, name string, sourceCode string, entry string, args []goja.Value) (string, error) {
	vm, err := k.buildContract(ctx, name, sourceCode, nil, true)
	if err != nil {
		return "", err
	}

	ctc := vm.Get("CONTRACT").ToObject(vm)
	if ctc == nil {
		return "", sdkerrors.Wrapf(sdkerrors.ErrKeyNotFound, "cannot find contract")
	}

	r := ctc.Get("queries").ToObject(vm)
	if r == nil {
		return "", sdkerrors.Wrapf(sdkerrors.ErrKeyNotFound, "cannot find queries")
	}

	function, ok := goja.AssertFunction(r.Get(entry))
	if !ok {
		return "", sdkerrors.Wrapf(sdkerrors.ErrKeyNotFound, "cannot get %s from contractQueries", entry)
	}

	res, err := function(goja.Undefined(), args...)
	if err != nil {
		return "", err
	}

	return res.String(), nil
}

func (k Keeper) buildContract(ctx sdk.Context, name string, sourceCode string, creator sdk.AccAddress, readonly bool) (*goja.Runtime, error) {
	vm := goja.New()
	ctx.GasMeter().ConsumeGas(types.DefaultGasValues().Init, "starting virtual machine")

	address, err := k.getContractAddress(ctx, name)
	if err != nil {
		return vm, err
	}

	stdlib.ApplyStandardLib(ctx, k, creator, name, address, vm, readonly)

	_, err = vm.RunString(sourceCode)
	if err != nil {
		return vm, err
	}

	return vm, nil
}
