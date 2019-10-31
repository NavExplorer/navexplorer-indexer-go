package explorer

type Proposal struct {
	Version             uint32 `json:"version"`
	Hash                string `json:"hash"`
	BlockHash           string `json:"blockHash"`
	Description         string `json:"description"`
	RequestedAmount     uint64 `json:"requestedAmount"`
	NotPaidYet          uint64 `json:"notPaidYet"`
	UserPaidFee         uint64 `json:"userPaidFee"`
	PaymentAddress      string `json:"paymentAddress"`
	ProposalDuration    uint64 `json:"proposalDuration"`
	ExpiresOn           uint64 `json:"expiresOn"`
	VotesYes            uint   `json:"votesYes"`
	VotesNo             uint   `json:"votesNo"`
	VotingCycle         uint   `json:"votingCycle"`
	Status              string `json:"status"`
	State               uint   `json:"state"`
	StateChangedOnBlock string `json:"stateChangedOnBlock,omitempty"`
}
