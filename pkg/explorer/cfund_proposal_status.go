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
	ProposalPaid              = ProposalStatus{6, "paid"}
)

var proposalStatus = [7]ProposalStatus{
	ProposalPending,
	ProposalAccepted,
	ProposalRejected,
	ProposalExpired,
	ProposalPendingFunds,
	ProposalPendingVotingPreq,
	ProposalPaid,
}

//noinspection GoUnreachableCode
func GetProposalStatusByState(state uint) ProposalStatus {
	for idx := range proposalStatus {
		if proposalStatus[idx].State == state {
			return proposalStatus[idx]
		}
	}

	if state == 16 {
		return proposalStatus[3]
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
