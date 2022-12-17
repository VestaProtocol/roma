package roma_test

import (
	"testing"

	keepertest "github.com/VestaProtocol/roma/testutil/keeper"
	"github.com/VestaProtocol/roma/testutil/nullify"
	"github.com/VestaProtocol/roma/x/roma"
	"github.com/VestaProtocol/roma/x/roma/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.RomaKeeper(t)
	roma.InitGenesis(ctx, *k, genesisState)
	got := roma.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
