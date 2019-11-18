package address

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/NavExplorer/navexplorer-indexer-go/internal/elastic_cache"
	"github.com/NavExplorer/navexplorer-indexer-go/pkg/explorer"
	"github.com/olivere/elastic/v7"
	"strings"
)

type Repository struct {
	Client *elastic.Client
}

func NewRepo(Client *elastic.Client) *Repository {
	return &Repository{Client}
}

func (r *Repository) GetAddresses(hashes []string) ([]*explorer.Address, error) {
	results, err := r.Client.Search(elastic_cache.AddressIndex.Get()).
		Query(elastic.NewMatchQuery("hash", strings.Join(hashes, " "))).
		Size(len(hashes)).
		Do(context.Background())
	if err != nil || results == nil {
		return nil, err
	}

	addresses := make([]*explorer.Address, 0)
	for _, hit := range results.Hits.Hits {
		var address *explorer.Address
		if err = json.Unmarshal(hit.Source, &address); err != nil {
			return nil, err
		}
		address.MetaData = explorer.NewMetaData(hit.Id, hit.Index)
		addresses = append(addresses, address)
	}

	return addresses, nil
}

func (r *Repository) GetAddressesHeightGt(height uint64) ([]string, error) {
	results, err := r.Client.
		Search(elastic_cache.AddressTransactionIndex.Get()).
		Query(elastic.NewRangeQuery("height").Gt(height)).
		Aggregation("hash", elastic.NewTermsAggregation().Field("hash.keyword").Size(50000000)).
		Size(0).
		Do(context.Background())
	if err != nil || results == nil {
		return nil, err
	}

	addresses := make([]string, 0)
	if agg, found := results.Aggregations.Terms("hash"); found {
		for _, bucket := range agg.Buckets {
			addresses = append(addresses, bucket.Key.(string))
		}
	}

	return addresses, nil
}

func (r *Repository) GetOrCreateAddress(hash string) (*explorer.Address, error) {
	results, err := r.Client.
		Search(elastic_cache.AddressIndex.Get()).
		Query(elastic.NewMatchQuery("hash", hash)).
		Size(1).
		Do(context.Background())
	if err != nil || results == nil {
		return nil, err
	}

	var address *explorer.Address
	if len(results.Hits.Hits) == 0 {
		address = CreateAddress(hash)
		resp, err := r.Client.Index().Index(elastic_cache.AddressIndex.Get()).BodyJson(address).Do(context.Background())
		if err != nil {
			return nil, err
		}
		address.MetaData = explorer.NewMetaData(resp.Id, resp.Index)

		return address, nil
	}

	if err = json.Unmarshal(results.Hits.Hits[0].Source, &address); err != nil {
		return nil, err
	}
	address.MetaData = explorer.NewMetaData(results.Hits.Hits[0].Id, results.Hits.Hits[0].Index)

	return address, nil
}

func (r *Repository) GetAddress(hash string) (*explorer.Address, error) {
	results, err := r.Client.
		Search(elastic_cache.AddressIndex.Get()).
		Query(elastic.NewTermQuery("hash", hash)).
		Size(1).
		Do(context.Background())
	if err != nil {
		return nil, err
	}

	if results == nil || len(results.Hits.Hits) != 1 {
		return nil, errors.New("Invalid result found")
	}

	var address *explorer.Address
	if err = json.Unmarshal(results.Hits.Hits[0].Source, &address); err != nil {
		return nil, err
	}

	return address, nil
}

func (r *Repository) GetTxsRangeForAddress(hash string, from uint64, to uint64) ([]*explorer.AddressTransaction, error) {
	query := elastic.NewBoolQuery()
	query = query.Must(elastic.NewMatchQuery("hash", hash))
	query = query.Must(elastic.NewRangeQuery("height").Gt(from).Lte(to))

	results, err := r.Client.
		Search(elastic_cache.AddressTransactionIndex.Get()).
		Query(query).
		Size(50000000).
		Sort("height", true).
		Do(context.Background())
	if err != nil {
		return nil, err
	}

	txs := make([]*explorer.AddressTransaction, 0)
	for _, hit := range results.Hits.Hits {
		var tx *explorer.AddressTransaction
		if err = json.Unmarshal(hit.Source, &tx); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}

	return txs, nil
}

func (r *Repository) GetTxsForAddress(hash string) ([]*explorer.AddressTransaction, error) {
	results, err := r.Client.
		Search(elastic_cache.AddressTransactionIndex.Get()).
		Query(elastic.NewMatchQuery("hash", hash)).
		Size(50000000).
		Sort("height", true).
		Do(context.Background())
	if err != nil {
		return nil, err
	}

	txs := make([]*explorer.AddressTransaction, 0)
	for _, hit := range results.Hits.Hits {
		var tx *explorer.AddressTransaction
		if err = json.Unmarshal(hit.Source, &tx); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}

	return txs, nil
}
