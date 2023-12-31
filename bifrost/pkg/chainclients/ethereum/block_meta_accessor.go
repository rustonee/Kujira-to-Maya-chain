package ethereum

import (
	"gitlab.com/mayachain/mayanode/bifrost/pkg/chainclients/ethereum/types"
)

// BlockMetaAccessor define methods need to access block meta storage
type BlockMetaAccessor interface {
	GetBlockMetas() ([]*types.BlockMeta, error)
	GetBlockMeta(height int64) (*types.BlockMeta, error)
	SaveBlockMeta(height int64, block *types.BlockMeta) error
	PruneBlockMeta(height int64) error
	AddSignedTxItem(item SignedTxItem) error
	RemoveSignedTxItem(hash string) error
	GetSignedTxItems() ([]SignedTxItem, error)
}
