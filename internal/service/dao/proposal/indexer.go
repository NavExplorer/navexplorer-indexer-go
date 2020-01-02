package proposal

import (
	"context"
	"github.com/NavExplorer/navcoind-go"
	"github.com/NavExplorer/navexplorer-indexer-go/internal/elastic_cache"
	"github.com/NavExplorer/navexplorer-indexer-go/pkg/explorer"
	log "github.com/sirupsen/logrus"
)

type Indexer struct {
	navcoin   *navcoind.Navcoind
	elastic   *elastic_cache.Index
	indexSize uint64
}

func NewIndexer(navcoin *navcoind.Navcoind, elastic *elastic_cache.Index, indexSize uint64) *Indexer {
	return &Indexer{navcoin, elastic, indexSize}
}

func (i *Indexer) Index(txs []*explorer.BlockTransaction) {
	for _, tx := range txs {
		if !tx.IsSpend() && tx.Version != 4 {
			continue
		}

		if navP, err := i.navcoin.GetProposal(tx.Hash); err == nil {
			proposal := CreateProposal(navP, tx.Height)

			resp, err := i.elastic.Client.Index().Index(elastic_cache.ProposalIndex.Get()).BodyJson(proposal).Do(context.Background())
			if err != nil {
				log.WithError(err).Fatal("Failed to save new proposal")
			}

			proposal.MetaData = explorer.NewMetaData(resp.Id, resp.Index)
			Proposals = append(Proposals, proposal)
		}
	}
}

func (i *Indexer) Update(blockCycle *explorer.BlockCycle, block *explorer.Block) {
	for _, p := range Proposals {
		if p == nil {
			continue
		}

		navP, err := i.navcoin.GetProposal(p.Hash)
		if err != nil {
			log.WithError(err).Fatalf("Failed to find active proposal: %s", p.Hash)
		}

		UpdateProposal(navP, block.Height, p)
		if p.UpdatedOnBlock == block.Height {
			i.elastic.AddUpdateRequest(elastic_cache.ProposalIndex.Get(), p.Hash, p, p.MetaData.Id)
		}

		if p.Status == explorer.ProposalExpired || p.Status == explorer.ProposalRejected {
			if block.Height-p.UpdatedOnBlock >= uint64(blockCycle.Size) {
				Proposals.Delete(p.Hash)
			}
		}
	}
}
