package testutil

import (
	"context"
	"testing"

	"github.com/baotoq/shitcoin/internal/domain/chain"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface compliance check.
var _ chain.Repository = (*MockChainRepo)(nil)

func TestMockChainRepo_SaveAndGetBlock(t *testing.T) {
	repo := NewMockChainRepo()
	ctx := context.Background()
	b := MustCreateBlock(t, 0, [32]byte{})

	err := repo.SaveBlock(ctx, b)
	require.NoError(t, err)

	// Retrieve by hash
	got, err := repo.GetBlock(ctx, b.Hash())
	require.NoError(t, err)
	assert.Equal(t, b.Hash(), got.Hash())

	// Retrieve by height
	got, err = repo.GetBlockByHeight(ctx, 0)
	require.NoError(t, err)
	assert.Equal(t, b.Hash(), got.Hash())
}

func TestMockChainRepo_SaveBlockWithUTXOs(t *testing.T) {
	repo := NewMockChainRepo()
	ctx := context.Background()
	b := MustCreateBlock(t, 0, [32]byte{})
	undo := &utxo.UndoEntry{BlockHeight: 0, Spent: nil, Created: nil}

	err := repo.SaveBlockWithUTXOs(ctx, b, undo)
	require.NoError(t, err)

	// Block should be retrievable
	got, err := repo.GetBlock(ctx, b.Hash())
	require.NoError(t, err)
	assert.Equal(t, b.Hash(), got.Hash())

	// Undo entry should be retrievable
	gotUndo, err := repo.GetUndoEntry(ctx, 0)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), gotUndo.BlockHeight)
}

func TestMockChainRepo_GetBlock_UnknownHash(t *testing.T) {
	repo := NewMockChainRepo()
	ctx := context.Background()

	_, err := repo.GetBlock(ctx, [32]byte{1, 2, 3})
	assert.Error(t, err)
}

func TestMockChainRepo_GetBlockByHeight_UnknownHeight(t *testing.T) {
	repo := NewMockChainRepo()
	ctx := context.Background()

	_, err := repo.GetBlockByHeight(ctx, 99)
	assert.Error(t, err)
}

func TestMockChainRepo_GetLatestBlock(t *testing.T) {
	repo := NewMockChainRepo()
	ctx := context.Background()

	genesis := MustCreateBlock(t, 0, [32]byte{})
	block1 := MustCreateBlock(t, 1, genesis.Hash())

	require.NoError(t, repo.SaveBlock(ctx, genesis))
	require.NoError(t, repo.SaveBlock(ctx, block1))

	latest, err := repo.GetLatestBlock(ctx)
	require.NoError(t, err)
	assert.Equal(t, block1.Hash(), latest.Hash(), "latest should be block at height 1")
}

func TestMockChainRepo_GetLatestBlock_Empty(t *testing.T) {
	repo := NewMockChainRepo()
	ctx := context.Background()

	_, err := repo.GetLatestBlock(ctx)
	assert.Error(t, err)
}

func TestMockChainRepo_GetChainHeight(t *testing.T) {
	repo := NewMockChainRepo()
	ctx := context.Background()

	// Empty chain
	height, err := repo.GetChainHeight(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), height)

	// After saving blocks
	genesis := MustCreateBlock(t, 0, [32]byte{})
	block1 := MustCreateBlock(t, 1, genesis.Hash())
	require.NoError(t, repo.SaveBlock(ctx, genesis))
	require.NoError(t, repo.SaveBlock(ctx, block1))

	height, err = repo.GetChainHeight(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), height)
}

func TestMockChainRepo_GetBlocksInRange(t *testing.T) {
	repo := NewMockChainRepo()
	ctx := context.Background()

	blocks := MustCreateBlockChain(t, 5)
	for _, b := range blocks {
		require.NoError(t, repo.SaveBlock(ctx, b))
	}

	// Get range [1, 3]
	result, err := repo.GetBlocksInRange(ctx, 1, 3)
	require.NoError(t, err)
	require.Len(t, result, 3)
	assert.Equal(t, uint64(1), result[0].Height())
	assert.Equal(t, uint64(2), result[1].Height())
	assert.Equal(t, uint64(3), result[2].Height())
}

func TestMockChainRepo_GetUndoEntry(t *testing.T) {
	repo := NewMockChainRepo()
	ctx := context.Background()
	b := MustCreateBlock(t, 0, [32]byte{})
	undo := &utxo.UndoEntry{BlockHeight: 0, Spent: nil, Created: nil}

	require.NoError(t, repo.SaveBlockWithUTXOs(ctx, b, undo))

	got, err := repo.GetUndoEntry(ctx, 0)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), got.BlockHeight)

	// Non-existent height
	_, err = repo.GetUndoEntry(ctx, 99)
	assert.Error(t, err)
}

func TestMockChainRepo_DeleteBlocksAbove(t *testing.T) {
	repo := NewMockChainRepo()
	ctx := context.Background()

	blocks := MustCreateBlockChain(t, 5)
	for _, b := range blocks {
		undo := &utxo.UndoEntry{BlockHeight: b.Height()}
		require.NoError(t, repo.SaveBlockWithUTXOs(ctx, b, undo))
	}

	// Delete blocks above height 2 (should remove heights 3, 4)
	err := repo.DeleteBlocksAbove(ctx, 2)
	require.NoError(t, err)

	// Heights 0-2 should still exist
	for i := uint64(0); i <= 2; i++ {
		_, err := repo.GetBlockByHeight(ctx, i)
		require.NoError(t, err, "block at height %d should still exist", i)
	}

	// Heights 3-4 should be gone
	for i := uint64(3); i <= 4; i++ {
		_, err := repo.GetBlockByHeight(ctx, i)
		assert.Error(t, err, "block at height %d should be deleted", i)
	}

	// Latest should now be block at height 2
	latest, err := repo.GetLatestBlock(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(2), latest.Height())

	// Chain height should be 2
	height, err := repo.GetChainHeight(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(2), height)
}

func TestMockChainRepo_AddBlock(t *testing.T) {
	repo := NewMockChainRepo()
	ctx := context.Background()
	b := MustCreateBlock(t, 0, [32]byte{})

	repo.AddBlock(b)

	got, err := repo.GetBlock(ctx, b.Hash())
	require.NoError(t, err)
	assert.Equal(t, b.Hash(), got.Hash())
}
