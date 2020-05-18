package block

import (
	"context"
	"encoding/json"
	"github.com/NavExplorer/navexplorer-indexer-go/internal/elastic_cache"
	"github.com/NavExplorer/navexplorer-indexer-go/pkg/explorer"
	"github.com/getsentry/raven-go"
	"github.com/olivere/elastic/v7"
)

type Repository struct {
	Client *elastic.Client
}

func NewRepo(Client *elastic.Client) *Repository {
	return &Repository{Client}
}

func (r *Repository) GetBestBlock() (*explorer.Block, error) {
	results, err := r.Client.
		Search(elastic_cache.BlockIndex.Get()).
		Sort("height", false).
		Size(1).
		Do(context.Background())
	if err != nil || results == nil {
		return nil, err
	}

	if len(results.Hits.Hits) == 0 {
		return nil, ErrBlockNotFound
	}

	var block *explorer.Block
	if err = json.Unmarshal(results.Hits.Hits[0].Source, &block); err != nil {
		return nil, err
	}

	return block, nil
}

func (r *Repository) GetHeight() (uint64, error) {
	b, err := r.GetBestBlock()
	if err != nil {
		return 0, err
	}

	return b.Height, nil
}

func (r *Repository) GetBlockByHash(hash string) (*explorer.Block, error) {
	results, err := r.Client.
		Search(elastic_cache.BlockIndex.Get()).
		Query(elastic.NewMatchQuery("hash", hash)).
		Size(1).
		Do(context.Background())
	if err != nil || results == nil {
		raven.CaptureError(err, nil)
		return nil, err
	}

	if len(results.Hits.Hits) == 0 {
		raven.CaptureError(err, nil)
		return nil, elastic_cache.ErrRecordNotFound
	}

	var block *explorer.Block
	hit := results.Hits.Hits[0]
	if err = json.Unmarshal(hit.Source, &block); err != nil {
		raven.CaptureError(err, nil)
		return nil, err
	}

	return block, nil
}

func (r *Repository) GetBlockByHeight(height uint64) (*explorer.Block, error) {
	results, err := r.Client.
		Search(elastic_cache.BlockIndex.Get()).
		Query(elastic.NewMatchQuery("height", height)).
		Size(1).
		Do(context.Background())
	if err != nil || results == nil {
		raven.CaptureError(err, nil)
		return nil, err
	}

	if len(results.Hits.Hits) == 0 {
		raven.CaptureError(err, nil)
		return nil, elastic_cache.ErrRecordNotFound
	}

	var block *explorer.Block
	hit := results.Hits.Hits[0]
	if err = json.Unmarshal(hit.Source, &block); err != nil {
		raven.CaptureError(err, nil)
		return nil, err
	}

	return block, nil
}
