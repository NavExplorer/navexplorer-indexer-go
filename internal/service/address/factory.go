package address

import (
	"encoding/json"
	"github.com/NavExplorer/navexplorer-indexer-go/pkg/explorer"
	log "github.com/sirupsen/logrus"
)

func CreateAddress(hash string) *explorer.Address {
	return &explorer.Address{Hash: hash}
}

func ResetAddress(address *explorer.Address) *explorer.Address {
	return &explorer.Address{MetaData: address.MetaData, Hash: address.Hash}
}

func ApplyTxToAddress(address *explorer.Address, tx *explorer.AddressTransaction) {
	address.Height = tx.Height

	if tx.Cold == true {
		address.ColdBalance += tx.Total
		if explorer.IsColdStake(tx.Type) {
			address.ColdStaked += tx.Total
			address.ColdStakedCount++
		} else if tx.Type == explorer.TransferSend {
			address.ColdSent += tx.Input
			address.ColdSentCount++
		} else if tx.Type == explorer.TransferReceive {
			address.ColdReceived += tx.Output
			address.ColdReceivedCount++
		}
	} else {
		address.Balance += tx.Total
		if explorer.IsStake(tx.Type) || explorer.IsColdStake(tx.Type) {
			address.Staked += tx.Total
			address.StakedCount++
		} else if tx.Type == explorer.TransferSend {
			address.Sent += tx.Input
			address.SentCount++
		} else if tx.Type == explorer.TransferReceive {
			address.Received += tx.Output
			address.ReceivedCount++
		} else if tx.Type == explorer.TransferPoolFee {
			address.Staked += tx.Total
			address.StakedCount++
		}
	}
}

func CreateAddressTransactions(txs []*explorer.BlockTransaction) []*explorer.AddressTransaction {
	if len(txs) == 0 {
		return nil
	}

	addressTxs := make([]*explorer.AddressTransaction, 0)
	for _, tx := range txs {
		for _, address := range tx.GetAllAddresses() {
			if tx.HasColdInput(address) || tx.HasColdStakeStake(address) || tx.HasColdStakeReceive(address) {
				if coldAddressTx := createColdTransaction(address, tx); coldAddressTx != nil {
					addressTxs = append(addressTxs, coldAddressTx)
				}
			}
			if addressTx := createTransaction(address, tx); addressTx != nil {
				addressTxs = append(addressTxs, addressTx)
			}
		}
	}

	return addressTxs
}

func createTransaction(address string, tx *explorer.BlockTransaction) *explorer.AddressTransaction {
	_, input := tx.Vin.GetAmountByAddress(address, false)
	_, output := tx.Vout.GetAmountByAddress(address, false)
	if input+output == 0 {
		return nil
	}

	addressTransaction := &explorer.AddressTransaction{
		Hash:   address,
		Txid:   tx.Hash,
		Height: tx.Height,
		Time:   tx.Time,
		Cold:   false,
		Input:  input,
		Output: output,
		Total:  output - input,
	}

	if tx.IsStaking() {
		if tx.Vin.HasAddress(address) {
			addressTransaction.Type = explorer.TransferStake
		} else {
			addressTransaction.Type = explorer.TransferDelegateStake
		}
	} else if tx.IsCoinbase() {
		if tx.Version == 1 || tx.Stake != 0 {
			// POW block_indexer
			addressTransaction.Type = explorer.TransferStake
		} else if tx.Version == 3 {
			addressTransaction.Type = explorer.TransferCommunityFundPayout
		} else {
			bt, _ := json.Marshal(tx)
			log.WithFields(log.Fields{"tx": string(bt)}).Fatal("Could not handle coinbase")
		}
	} else if tx.IsColdStaking() {
		if addressTransaction.Input == addressTransaction.Output {
			addressTransaction.Type = explorer.TransferStake
		} else if tx.HasColdStakeSpend(address) {
			if tx.Vin.HasAddress(address) {
				addressTransaction.Type = explorer.TransferColdStake
			} else {
				addressTransaction.Type = explorer.TransferColdDelegateStake
			}
		} else if !tx.Vin.HasAddress(address) && tx.Vout.HasAddress(address) {
			addressTransaction.Type = explorer.TransferPoolFee
		} else if addressTransaction.Input == 0 {
			addressTransaction.Type = explorer.TransferReceive
		}
	} else {
		if addressTransaction.Input > addressTransaction.Output {
			addressTransaction.Type = explorer.TransferSend
		} else {
			addressTransaction.Type = explorer.TransferReceive
		}
	}

	if addressTransaction.Type == "" {
		bt, _ := json.Marshal(tx)
		log.WithFields(log.Fields{"tx": string(bt)}).Fatal("addressTransaction.Type not identified: ", address)
		return nil
	}

	return addressTransaction
}

func createColdTransaction(address string, tx *explorer.BlockTransaction) *explorer.AddressTransaction {
	_, input := tx.Vin.GetAmountByAddress(address, true)
	_, output := tx.Vout.GetAmountByAddress(address, true)
	if input+output == 0 {
		return nil
	}

	addressTransaction := &explorer.AddressTransaction{
		Hash:   address,
		Txid:   tx.Hash,
		Height: tx.Height,
		Time:   tx.Time,
		Cold:   true,
		Input:  input,
		Output: output,
		Total:  output - input,
	}

	if tx.IsSpend() {
		if addressTransaction.Input > addressTransaction.Output {
			addressTransaction.Type = explorer.TransferSend
		} else {
			addressTransaction.Type = explorer.TransferReceive
		}
	} else if tx.IsColdStaking() {
		if tx.Vin.HasAddress(address) {
			addressTransaction.Type = explorer.TransferColdStake
		} else {
			addressTransaction.Type = explorer.TransferColdDelegateStake
		}
	} else {
		log.WithFields(log.Fields{"tx.type": tx.Type}).Fatal("WE FOUND SOMETHING ELSE IN COLD TX")
	}

	return addressTransaction
}
