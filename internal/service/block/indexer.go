package block

import (
	"github.com/NavExplorer/navcoind-go"
	"github.com/NavExplorer/navexplorer-indexer-go/internal/elastic_cache"
	"github.com/NavExplorer/navexplorer-indexer-go/pkg/explorer"
	log "github.com/sirupsen/logrus"
)

type Indexer struct {
	navcoin       *navcoind.Navcoind
	elastic       *elastic_cache.Index
	orphanService *OrphanService
}

func NewIndexer(navcoin *navcoind.Navcoind, elastic *elastic_cache.Index, orphanService *OrphanService) *Indexer {
	return &Indexer{navcoin, elastic, orphanService}
}

func (i *Indexer) Index(height uint64, option int) (*explorer.Block, []*explorer.BlockTransaction, error) {
	navBlock, err := i.getBlockAtHeight(height)
	if err != nil {
		return nil, nil, err
	}

	block := CreateBlock(navBlock)
	if option == 1 {
		if orphan, err := i.orphanService.IsOrphanBlock(block); orphan == true || err != nil {
			return nil, nil, ErrOrphanBlockFound
		}
	}

	var txs = make([]*explorer.BlockTransaction, 0)
	for idx, txHash := range block.Tx {
		rawTx, err := i.navcoin.GetRawTransaction(txHash, true)
		if err != nil {
			log.WithFields(log.Fields{"hash": block.Hash, "txHash": txHash, "height": height}).WithError(err).Error("Failed to GetRawTransaction")
			return nil, nil, err
		}
		tx := CreateBlockTransaction(rawTx.(navcoind.RawTransaction), uint(idx))
		applyType(tx)
		applyStaking(tx, block)
		applySpend(tx, block)
		applyCFundPayout(tx, block)
		i.indexPreviousTxData(*tx)

		txs = append(txs, tx)
		i.elastic.AddIndexRequest(elastic_cache.BlockTransactionIndex.Get(), tx.Hash, tx)
	}

	i.elastic.AddIndexRequest(elastic_cache.BlockIndex.Get(), block.Hash, block)

	return block, txs, err
}

func (i *Indexer) indexPreviousTxData(tx explorer.BlockTransaction) {
	vin := tx.Vin

	for vdx := range vin {
		if vin[vdx].Vout == nil || vin[vdx].Txid == nil {
			continue
		}

		rawTx, err := i.navcoin.GetRawTransaction(*vin[vdx].Txid, true)
		if err != nil {
			log.WithFields(log.Fields{"hash": *vin[vdx].Txid}).WithError(err).Fatal("Failed to get previous transaction")
		}

		prevTx := CreateBlockTransaction(rawTx.(navcoind.RawTransaction), 0)
		if len(prevTx.Vout) <= *vin[vdx].Vout {
			log.WithFields(log.Fields{"index": vdx, "tx": prevTx.Hash}).Fatal("Vout does not exist")
		}

		previousOutput := prevTx.Vout[*vin[vdx].Vout]
		vin[vdx].Value = previousOutput.Value
		vin[vdx].ValueSat = previousOutput.ValueSat
		vin[vdx].Addresses = previousOutput.ScriptPubKey.Addresses
		vin[vdx].PreviousOutput.Type = previousOutput.ScriptPubKey.Type
		vin[vdx].PreviousOutput.Height = prevTx.Height
	}

	i.elastic.AddIndexRequest(elastic_cache.BlockTransactionIndex.Get(), tx.Hash, tx)
}

func (i *Indexer) getBlockAtHeight(height uint64) (*navcoind.Block, error) {
	hash, err := i.navcoin.GetBlockHash(height)
	if err != nil {
		if err.Error() != "-8: Block height out of range" {
			log.WithFields(log.Fields{"hash": hash, "height": height}).WithError(err).Error("Failed to GetBlockHash")
		}
		return nil, err
	}

	block, err := i.navcoin.GetBlock(hash)
	if err != nil {
		log.WithFields(log.Fields{"hash": hash, "height": height}).WithError(err).Error("Failed to GetBlock")
		return nil, err
	}

	return &block, nil
}
