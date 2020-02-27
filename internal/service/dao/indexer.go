package dao

import (
	"github.com/NavExplorer/navcoind-go"
	"github.com/NavExplorer/navexplorer-indexer-go/internal/service/dao/consensus"
	"github.com/NavExplorer/navexplorer-indexer-go/internal/service/dao/payment_request"
	"github.com/NavExplorer/navexplorer-indexer-go/internal/service/dao/proposal"
	"github.com/NavExplorer/navexplorer-indexer-go/internal/service/dao/vote"
	"github.com/NavExplorer/navexplorer-indexer-go/pkg/explorer"
	"github.com/getsentry/raven-go"
	log "github.com/sirupsen/logrus"
)

type Indexer struct {
	proposalIndexer       *proposal.Indexer
	paymentRequestIndexer *payment_request.Indexer
	voteIndexer           *vote.Indexer
	consensusIndexer      *consensus.Indexer
	navcoin               *navcoind.Navcoind
}

func NewIndexer(
	proposalIndexer *proposal.Indexer,
	paymentRequestIndexer *payment_request.Indexer,
	voteIndexer *vote.Indexer,
	consensusIndexer *consensus.Indexer,
	navcoin *navcoind.Navcoind,
) *Indexer {
	return &Indexer{
		proposalIndexer,
		paymentRequestIndexer,
		voteIndexer,
		consensusIndexer,
		navcoin,
	}
}

func (i *Indexer) Index(block *explorer.Block, txs []*explorer.BlockTransaction) {
	if consensus.Consensus == nil {
		err := i.consensusIndexer.Index()
		if err != nil {
			raven.CaptureError(err, nil)
			log.WithError(err).Fatal("Failed to get Consensus")
		}
	}

	header, err := i.navcoin.GetBlockheader(block.Hash)
	if err != nil {
		raven.CaptureError(err, nil)
		log.WithError(err).Fatal("Failed to get blockHeader")
	}

	blockCycle := block.BlockCycle(consensus.Consensus.BlocksPerVotingCycle, consensus.Consensus.MinSumVotesPerVotingCycle)

	i.proposalIndexer.Index(txs)
	i.paymentRequestIndexer.Index(txs)
	i.voteIndexer.IndexVotes(txs, block, header)

	if blockCycle.IsEnd() {
		log.WithFields(log.Fields{"Quorum": blockCycle.Quorum, "height": block.Height}).Debug("Dao - End of voting cycle")
		i.proposalIndexer.Update(blockCycle, block)
		i.paymentRequestIndexer.Update(blockCycle, block)
		_ = i.consensusIndexer.Index()
	}
}
