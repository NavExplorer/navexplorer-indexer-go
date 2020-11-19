package block

import (
	"fmt"
	"github.com/NavExplorer/navcoind-go"
	"github.com/NavExplorer/navexplorer-indexer-go/v2/internal/service/softfork"
	"github.com/NavExplorer/navexplorer-indexer-go/v2/pkg/explorer"
	"github.com/sirupsen/logrus"
	"time"
)

func CreateBlock(block *navcoind.Block, previousBlock *explorer.Block, cycleSize uint) *explorer.Block {
	logrus.Debugf("Create Block %s", block.Hash)
	return &explorer.Block{
		RawBlock: explorer.RawBlock{
			Hash:              block.Hash,
			Confirmations:     block.Confirmations,
			StrippedSize:      block.StrippedSize,
			Size:              block.Size,
			Weight:            block.Weight,
			Height:            block.Height,
			Version:           block.Version,
			VersionHex:        block.VersionHex,
			Merkleroot:        block.MerkleRoot,
			Tx:                block.Tx,
			Time:              time.Unix(block.Time, 0),
			MedianTime:        time.Unix(block.MedianTime, 0),
			Nonce:             block.Nonce,
			Bits:              block.Bits,
			Difficulty:        fmt.Sprintf("%f", block.Difficulty),
			Chainwork:         block.ChainWork,
			Previousblockhash: block.PreviousBlockHash,
			Nextblockhash:     block.NextBlockHash,
		},
		BlockCycle: createBlockCycle(cycleSize, previousBlock),
		TxCount:    uint(len(block.Tx)),
	}
}

func createBlockCycle(size uint, previousBlock *explorer.Block) *explorer.BlockCycle {
	if previousBlock == nil {
		return &explorer.BlockCycle{
			Size:  size,
			Cycle: 1,
			Index: 1,
		}
	}

	if !previousBlock.BlockCycle.IsEnd() {
		return &explorer.BlockCycle{
			Size:  size,
			Cycle: previousBlock.BlockCycle.Cycle,
			Index: previousBlock.BlockCycle.Index + 1,
		}
	}

	bc := &explorer.BlockCycle{
		Size:  size,
		Cycle: previousBlock.BlockCycle.Cycle + 1,
		Index: uint(previousBlock.Height+1) % size,
	}
	if bc.Index != 0 {
		bc.Transitory = true
		bc.TransitorySize = size - bc.Index
	}

	return bc
}

func CreateBlockTransaction(rawTx navcoind.RawTransaction, index uint) *explorer.BlockTransaction {
	tx := &explorer.BlockTransaction{
		RawBlockTransaction: explorer.RawBlockTransaction{
			Hex:             rawTx.Hex,
			Txid:            rawTx.Txid,
			Hash:            rawTx.Hash,
			Size:            rawTx.Size,
			VSize:           rawTx.VSize,
			Version:         rawTx.Version,
			LockTime:        rawTx.LockTime,
			Strdzeel:        rawTx.Strdzeel,
			AnonDestination: rawTx.AnonDestination,
			BlockHash:       rawTx.BlockHash,
			Height:          rawTx.Height,
			Confirmations:   rawTx.Confirmations,
			Time:            time.Unix(rawTx.Time, 0),
			BlockTime:       time.Unix(rawTx.BlockTime, 0),
		},
		Index: index,
		Vin:   createVin(rawTx.Vin),
		Vout:  createVout(rawTx.Vout),
	}

	return tx
}

func createVin(vins []navcoind.Vin) []explorer.Vin {
	var inputs = make([]explorer.Vin, 0)
	for idx, _ := range vins {
		input := explorer.Vin{
			RawVin: explorer.RawVin{
				Coinbase: vins[idx].Coinbase,
				Sequence: vins[idx].Sequence,
			},
		}
		if vins[idx].Txid != "" {
			input.Txid = &vins[idx].Txid
			input.Vout = &vins[idx].Vout
		}

		if vins[idx].Value != 0 {
			input.Value = vins[idx].Value
			input.ValueSat = vins[idx].ValueSat
		}

		if vins[idx].Address != "" {
			input.Addresses = []string{vins[idx].Address}
		}

		inputs = append(inputs, input)
	}

	return inputs
}

func createVout(vouts []navcoind.Vout) []explorer.Vout {
	var output = make([]explorer.Vout, 0)
	for _, o := range vouts {
		output = append(output, explorer.Vout{
			RawVout: explorer.RawVout{
				Value:    o.Value,
				ValueSat: o.ValueSat,
				N:        o.N,
				ScriptPubKey: explorer.ScriptPubKey{
					Asm:       o.ScriptPubKey.Asm,
					Hex:       o.ScriptPubKey.Hex,
					ReqSigs:   o.ScriptPubKey.ReqSigs,
					Type:      explorer.VoutTypes[o.ScriptPubKey.Type],
					Addresses: o.ScriptPubKey.Addresses,
					Hash:      o.ScriptPubKey.Hash,
				},
				SpendingKey:  o.SpendingKey,
				OutputKey:    o.OutputKey,
				EphemeralKey: o.EphemeralKey,
				RangeProof:   o.RangeProof,
				SpentTxId:    o.SpentTxId,
				SpentIndex:   o.SpentIndex,
				SpentHeight:  uint64(o.SpentHeight),
			},
		})
	}

	return output
}

func applyType(tx *explorer.BlockTransaction) {
	if tx.IsCoinbase() {
		tx.Type = explorer.TxCoinbase
	} else if isStakingTx(tx) {
		tx.Type = explorer.TxStaking
	} else if tx.Vout.OutputAtIndexIsOfType(1, explorer.VoutColdStaking) {
		tx.Type = explorer.TxColdStaking
	} else if tx.Vout.OutputAtIndexIsOfType(1, explorer.VoutColdStakingV2) {
		tx.Type = explorer.TxColdStakingV2
	} else {
		tx.Type = explorer.TxSpend
	}
}

func isStakingTx(tx *explorer.BlockTransaction) bool {
	return tx.Vout.OutputAtIndexIsOfType(0, explorer.VoutNonstandard) &&
		tx.Vout.GetOutput(0).ScriptPubKey.Hex == ""
}

func applyPrivateStatus(tx *explorer.BlockTransaction) {
	if tx.IsCoinbase() {
		return
	}

	var idx int
	for idx = range tx.Vin {
		tx.Vin[idx].Private = tx.Vin[idx].PreviousOutput.Type == explorer.VoutNonstandard && len(tx.Vin[idx].Addresses) == 0
		tx.Private = true
	}
	for idx = range tx.Vout {
		tx.Vout[idx].Private = tx.Vout[idx].RangeProof
		tx.Private = true
	}
}

func applyStaking(tx *explorer.BlockTransaction, block *explorer.Block) {
	if tx.IsSpend() {
		return
	}

	if tx.IsAnyStaking() {
		if softfork.SoftForks.StaticRewards().IsActive() {
			tx.Stake = 200000000 // hard coded to 2 as static rewards arrived after block_indexer 2761920
			block.Stake += tx.Stake
		} else {
			tx.Stake = tx.Vout.GetSpendableAmount() - tx.Vin.GetAmount()
			block.Stake += tx.Stake
		}
	} else if tx.IsCoinbase() {
		for _, o := range tx.Vout {
			if o.ScriptPubKey.Type == explorer.VoutPubkey {
				tx.Stake = o.ValueSat
				block.Stake = o.ValueSat
			}
		}
	}

	voutsWithAddresses := tx.Vout.FilterWithAddresses()
	vinsWithAddresses := tx.Vin.FilterWithAddresses()

	if tx.IsColdStaking() {
		for _, vout := range tx.Vout {
			if vout.ScriptPubKey.Type == explorer.VoutColdStaking {
				block.StakedBy = vout.ScriptPubKey.Addresses[0]
				break
			}
		}
		block.StakedBy = voutsWithAddresses[0].ScriptPubKey.Addresses[0]
	} else if len(vinsWithAddresses) != 0 {
		block.StakedBy = vinsWithAddresses[0].Addresses[0]
	} else if len(voutsWithAddresses) != 0 {
		block.StakedBy = voutsWithAddresses[0].ScriptPubKey.Addresses[0]
	}
}

func applySpend(tx *explorer.BlockTransaction, block *explorer.Block) {
	if tx.Type != explorer.TxSpend {
		return
	}

	tx.Spend = tx.Vout.GetAmount() - tx.Fees
	block.Spend += tx.Spend
}

func applyFees(tx *explorer.BlockTransaction, block *explorer.Block) {
	if tx.Type != explorer.TxSpend {
		return
	}

	if tx.Private == true {
		tx.Fees = tx.Vout.PrivateFees()
	} else {
		tx.Fees = tx.Vin.GetAmount() - tx.Vout.GetAmount()
	}
	block.Fees += tx.Fees
}

func applyCFundPayout(tx *explorer.BlockTransaction, block *explorer.Block) {
	if tx.IsCoinbase() {
		for _, o := range tx.Vout {
			if o.ScriptPubKey.Type == explorer.VoutPubkeyhash && tx.Version == 3 {
				block.CFundPayout += o.ValueSat
			}
		}
	}
}
