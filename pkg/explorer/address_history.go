package explorer

import (
	"fmt"
	"github.com/gosimple/slug"
	"time"
)

type AddressHistory struct {
	Height  uint64         `json:"height"`
	TxIndex uint           `json:"txindex"`
	Time    time.Time      `json:"time"`
	TxId    string         `json:"txid"`
	Hash    string         `json:"hash"`
	Changes AddressChanges `json:"changes"`
	Balance AddressBalance `json:"balance"`

	Stake       bool `json:"is_stake"`
	CfundPayout bool `json:"is_cfund_payout"`
	StakePayout bool `json:"is_stake_payout"`
}

type AddressChanges struct {
	Spending       int64 `json:"spending"`
	Staking        int64 `json:"staking"`
	Voting         int64 `json:"voting"`
	Proposal       bool  `json:"proposal,omitempty"`
	PaymentRequest bool  `json:"payment_request,omitempty"`
	Consultation   bool  `json:"consultation,omitempty"`
}

type AddressBalance struct {
	Spending int64 `json:"spending"`
	Staking  int64 `json:"staking"`
	Voting   int64 `json:"voting"`
}

type BalanceType string

var (
	Spending BalanceType = "spending"
	Staking  BalanceType = "staking"
	Voting   BalanceType = "voting"
)

func (a *AddressHistory) Slug() string {
	return slug.Make(fmt.Sprintf("addresshistory-%s-%s", a.Hash, a.TxId))
}

func (a *AddressHistory) IsSpend() bool {
	return a.Changes.Spending < 0
}

func (a *AddressHistory) IsReceive() bool {
	return a.Changes.Spending > 0
}