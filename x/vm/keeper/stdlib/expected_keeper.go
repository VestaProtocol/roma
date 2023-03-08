package stdlib

import (
	"github.com/VestaProtocol/roma/x/vm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dop251/goja"
)

type Keeper interface {
	InitFloats(ctx sdk.Context, vm *goja.Runtime, std *goja.Object)
	Injectors() []types.Injector
	SetRomdata(ctx sdk.Context, romdata types.Romdata)
	GetRomdata(ctx sdk.Context, index string) (val types.Romdata, found bool)
	RemoveRomdata(
		ctx sdk.Context,
		index string,
	)
	GetProgram(
		ctx sdk.Context,
		name string,
	) (val types.Program, found bool)
	GetContracts(ctx sdk.Context, id uint64) (val types.Contracts, found bool)
	QueryContract(ctx sdk.Context, name string, sourceCode string, entry string, args []goja.Value) (string, error)
	ExecuteContract(ctx sdk.Context, name string, sourceCode string, entry string, creator sdk.AccAddress, args []goja.Value) (string, error)
	GetContractVersion(program types.Program, version string) uint64
}
