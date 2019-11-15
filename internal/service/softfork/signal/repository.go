package signal

import (
	"context"
	"encoding/json"
	"github.com/NavExplorer/navexplorer-indexer-go/internal/elastic_cache"
	"github.com/NavExplorer/navexplorer-indexer-go/pkg/explorer"
	"github.com/olivere/elastic/v7"
)

type Repository struct {
	Client *elastic.Client
}

func NewRepo(client *elastic.Client) *Repository {
	return &Repository{client}
}

func (r *Repository) GetSignals(start uint64, end uint64) []*explorer.Signal {
	signals := make([]*explorer.Signal, 0)

	results, err := r.Client.Search(elastic_cache.SignalIndex.Get()).
		Sort("height", true).
		Query(elastic.NewRangeQuery("height").Gt(start).Lte(end)).
		Size(int(end - start)).
		Do(context.Background())

	if err == nil && results != nil && results.Hits != nil {
		for _, hit := range results.Hits.Hits {
			var signal *explorer.Signal
			if err := json.Unmarshal(hit.Source, &signal); err == nil {
				signals = append(signals, signal)
			}
		}
	}

	return signals
}
