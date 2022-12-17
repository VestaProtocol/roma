package keeper

import (
	"github.com/VestaProtocol/roma/x/vm/types"
)

var _ types.QueryServer = Keeper{}
