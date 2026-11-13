package v2

import (
	"errors"

	corestoretypes "cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MigrateStore migrates the x/wasm module state from consensus version 2 to 3.
// That migration copied parameters out of the legacy x/params subspace. SDK
// v0.55 removes x/params, and Gonka mainnet is already on x/wasm consensus
// version 4, so this path must not run.
func MigrateStore(_ sdk.Context, _ corestoretypes.KVStoreService, _ codec.BinaryCodec) error {
	return errors.New("x/wasm v2->v3 migration requires removed x/params module")
}
