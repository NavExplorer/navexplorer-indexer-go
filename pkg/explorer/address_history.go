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
	Reward  AddressReward  `json:"reward,omitempty"`

	Stake       bool `json:"is_stake"`
	ColdStake   bool `json:"is_coldstake"`
	CfundPayout bool `json:"is_cfund_payout"`
	StakePayout bool `json:"is_stake_payout"`
	MultiSig    bool `json:"is_multisig"`
	Order       uint `json:"order"`
}

type AddressChanges struct {
	Spendable      int64 `json:"spendable"`
	Stakable       int64 `json:"stakable"`
	VotingWeight   int64 `json:"voting_weight"`
	Proposal       bool  `json:"proposal,omitempty"`
	PaymentRequest bool  `json:"payment_request,omitempty"`
	Consultation   bool  `json:"consultation,omitempty"`
}

type AddressBalance struct {
	Spendable    int64 `json:"spendable"`
	Stakable     int64 `json:"stakable"`
	VotingWeight int64 `json:"voting_weight"`
}

type AddressReward struct {
	Spendable    float64 `json:"spendable"`
	Stakable     float64 `json:"stakable"`
	VotingWeight float64 `json:"voting_weight"`
}

type BalanceType string

var (
	Spendable    BalanceType = "spendable"
	Stakable     BalanceType = "stakable"
	VotingWeight BalanceType = "voting_weight"
)

func (a AddressHistory) Slug() string {
	return slug.Make(fmt.Sprintf("addresshistory-%s-%s", a.Hash, a.TxId))
}

func (a *AddressHistory) IsSpend() bool {
	return a.Changes.Spendable < 0
}

func (a *AddressHistory) IsReceive() bool {
	return a.Changes.Spendable > 0
}
