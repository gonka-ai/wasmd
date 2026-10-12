package keeper

import (
	"testing"

	storetypes "cosmossdk.io/store/types"
	wasmvm "github.com/CosmWasm/wasmvm/v2"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"
	"github.com/stretchr/testify/require"

	"github.com/CosmWasm/wasmd/x/wasm/keeper/wasmtesting"
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

func TestReplyOnlyRecursionIsBoundedAt500(t *testing.T) {
	require.Equal(t, uint32(500), types.DefaultMaxCallDepth)

	mock := &wasmtesting.MockWasmEngine{}
	wasmtesting.MakeInstantiable(mock)
	ctx, keepers := CreateTestInput(t, false, AvailableCapabilities, WithWasmEngine(mock))
	ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	instance := SeedNewContractInstance(t, ctx, keepers, mock)
	payee := instance.CreatorAddr.String()

	t.Run("depth 10 commits", func(t *testing.T) {
		replies := 0
		mock.ExecuteFn = func(wasmvm.Checksum, wasmvmtypes.Env, wasmvmtypes.MessageInfo, []byte, wasmvm.KVStore, wasmvm.GoAPI, wasmvm.Querier, wasmvm.GasMeter, uint64, wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
			return replyChain(payee, 10), 0, nil
		}
		mock.ReplyFn = func(wasmvm.Checksum, wasmvmtypes.Env, wasmvmtypes.Reply, wasmvm.KVStore, wasmvm.GoAPI, wasmvm.Querier, wasmvm.GasMeter, uint64, wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
			replies++
			if replies >= 10 {
				return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{}}, 0, nil
			}
			return replyChain(payee, 10-uint64(replies)), 0, nil
		}
		_, err := keepers.ContractKeeper.Execute(ctx, instance.Contract, instance.CreatorAddr, []byte(`{"recurse":{"depth":10}}`), nil)
		require.NoError(t, err)
		require.Equal(t, 10, replies)
	})

	t.Run("depth 20000 and 50000 stop at 500", func(t *testing.T) {
		for _, depth := range []uint64{20000, 50000} {
			replies := 0
			mock.ExecuteFn = func(wasmvm.Checksum, wasmvmtypes.Env, wasmvmtypes.MessageInfo, []byte, wasmvm.KVStore, wasmvm.GoAPI, wasmvm.Querier, wasmvm.GasMeter, uint64, wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
				return replyChain(payee, depth), 0, nil
			}
			mock.ReplyFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, reply wasmvmtypes.Reply, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
				replies++
				next := uint64(0)
				if reply.ID > 0 {
					next = reply.ID - 1
				}
				return replyChain(payee, next), 0, nil
			}
			_, err := keepers.ContractKeeper.Execute(ctx, instance.Contract, instance.CreatorAddr, []byte(`{}`), nil)
			require.ErrorIs(t, err, types.ErrExceedMaxCallDepth)
			require.Equal(t, int(types.DefaultMaxCallDepth), replies)
		}
	})
}

func replyChain(payee string, id uint64) *wasmvmtypes.ContractResult {
	return &wasmvmtypes.ContractResult{
		Ok: &wasmvmtypes.Response{
			Messages: []wasmvmtypes.SubMsg{{
				ID:      id,
				ReplyOn: wasmvmtypes.ReplyAlways,
				Msg: wasmvmtypes.CosmosMsg{Bank: &wasmvmtypes.BankMsg{Send: &wasmvmtypes.SendMsg{
					ToAddress: payee,
					Amount:    []wasmvmtypes.Coin{{Denom: "denom", Amount: "1"}},
				}}},
			}},
		},
	}
}
