package keeper_test

import (
	"testing"

	testkeeper "github.com/VestaProtocol/roma/testutil/keeper"
	"github.com/VestaProtocol/roma/x/roma/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.RomaKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
}
