package explorer

import (
	log "github.com/sirupsen/logrus"
)

type ProposalStatus struct {
	State  uint
	Status string
}

var (
	ProposalPending           = ProposalStatus{0, "pending"}
	ProposalAccepted          = ProposalStatus{1, "accepted"}
	ProposalRejected          = ProposalStatus{2, "rejected"}
	ProposalExpired           = ProposalStatus{3, "expired"}
	ProposalPendingFunds      = ProposalStatus{4, "pending_funds"}
	ProposalPendingVotingPreq = ProposalStatus{5, "pending_voting_preq"}
	ProposalAcceptedExpired   = ProposalStatus{16, "accepted_expired"}
)

var proposalStatus = [7]ProposalStatus{
	ProposalPending,
	ProposalAccepted,
	ProposalRejected,
	ProposalExpired,
	ProposalPendingFunds,
	ProposalPendingVotingPreq,
	ProposalAcceptedExpired,
}

//noinspection GoUnreachableCode
func GetProposalStatusByState(state uint) ProposalStatus {
	for idx := range proposalStatus {
		if proposalStatus[idx].State == state {
			return proposalStatus[idx]
		}
	}

	log.Fatal("ProposalStatus state does not exist", state)
	panic(0)
}

//noinspection GoUnreachableCode
func GetProposalStatusByStatus(status string) ProposalStatus {
	for idx := range proposalStatus {
		if proposalStatus[idx].Status == status {
			return proposalStatus[idx]
		}
	}

	log.Fatal("ProposalStatus status does not exist", status)
	panic(0)
}

func IsProposalStatusValid(status string) bool {
	for idx := range proposalStatus {
		if proposalStatus[idx].Status == status {
			return true
		}
	}
	return false
}

func IsProposalStateValid(state uint) bool {
	for idx := range proposalStatus {
		if proposalStatus[idx].State == state {
			return true
		}
	}
	return false
}
