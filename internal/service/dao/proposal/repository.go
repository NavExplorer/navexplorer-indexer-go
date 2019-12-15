package proposal

import (
	"context"
	"encoding/json"
	"github.com/NavExplorer/navexplorer-indexer-go/internal/elastic_cache"
	"github.com/NavExplorer/navexplorer-indexer-go/pkg/explorer"
	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"
)

type Repository struct {
	Client *elastic.Client
}

func NewRepo(client *elastic.Client) *Repository {
	return &Repository{client}
}

func (r *Repository) GetPossibleVotingProposals(height uint64) ([]*explorer.Proposal, error) {
	var proposals []*explorer.Proposal

	query := elastic.NewBoolQuery()
	query = query.Should(elastic.NewMatchQuery("status", "pending accepted"))
	query = query.Should(elastic.NewRangeQuery("updatedOnBlock").Gte(height))

	results, err := r.Client.Search(elastic_cache.ProposalIndex.Get()).
		Query(query).
		Size(9999).
		Sort("updatedOnBlock", false).
		Do(context.Background())
	if err != nil {
		return nil, err
	}
	if results == nil {
		return nil, elastic_cache.ErrResultsNotFound
	}

	for _, hit := range results.Hits.Hits {
		var proposal *explorer.Proposal
		if err := json.Unmarshal(hit.Source, &proposal); err != nil {
			log.WithError(err).Fatal("Failed to unmarshall proposal")
		}
		proposal.MetaData = explorer.NewMetaData(hit.Id, hit.Index)
		proposals = append(proposals, proposal)
	}

	return proposals, nil
}
