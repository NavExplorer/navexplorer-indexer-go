package elastic_cache

import (
	"context"
	"errors"
	"fmt"
	"github.com/NavExplorer/navexplorer-indexer-go/internal/config"
	"github.com/getsentry/raven-go"
	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Index struct {
	Client        *elastic.Client
	requests      []*Request
	bulkIndexSize uint
}

type Request struct {
	Height uint64
	Index  string
	Id     string
	Doc    interface{}
	Type   RequestType
}

type RequestType string

var (
	IndexRequest  RequestType = "index"
	UpdateRequest RequestType = "update"
)

var (
	ErrResultsNotFound = errors.New("Results not found")
	ErrRecordNotFound  = errors.New("Record not found")
)

func New() (*Index, error) {
	opts := []elastic.ClientOptionFunc{
		elastic.SetURL(strings.Join(config.Get().ElasticSearch.Hosts, ",")),
		elastic.SetSniff(config.Get().ElasticSearch.Sniff),
		elastic.SetHealthcheck(config.Get().ElasticSearch.HealthCheck),
	}

	if config.Get().ElasticSearch.Username != "" {
		opts = append(opts, elastic.SetBasicAuth(
			config.Get().ElasticSearch.Username,
			config.Get().ElasticSearch.Password,
		))
	}

	if config.Get().ElasticSearch.Debug {
		opts = append(opts, elastic.SetTraceLog(log.New(os.Stdout, "", 0)))
	}

	client, err := elastic.NewClient(opts...)
	if err != nil {
		log.Println("Error: ", err)
	}

	return &Index{
		Client:        client,
		requests:      make([]*Request, 0),
		bulkIndexSize: config.Get().BulkIndexSize,
	}, err
}

func (i *Index) InstallMappings() {
	logrus.Info("Install Mappings")
	files, err := ioutil.ReadDir(config.Get().ElasticSearch.MappingDir)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize ES")
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		b, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", config.Get().ElasticSearch.MappingDir, f.Name()))
		if err != nil {
			logrus.WithError(err).Fatal("Failed to initialize ES")
		}

		index := fmt.Sprintf("%s.%s", config.Get().Network, f.Name()[0:len(f.Name())-len(filepath.Ext(f.Name()))])
		if err = i.createIndex(index, b); err != nil {
			logrus.WithError(err).Fatal("Failed to initialize ES")
		}
	}
}

func (i *Index) AddIndexRequest(index string, id string, doc interface{}) {
	i.AddRequest(index, id, doc, IndexRequest)
}

func (i *Index) AddUpdateRequest(index string, id string, doc interface{}) {
	i.AddRequest(index, id, doc, UpdateRequest)
}

func (i *Index) AddRequest(index string, id string, doc interface{}, reqType RequestType) {
	if request := i.GetRequest(index, id); request != nil {
		request.Doc = doc
	} else {
		i.requests = append(i.requests, &Request{
			Index: index,
			Id:    id,
			Doc:   doc,
			Type:  reqType,
		})
	}
}

func (i *Index) GetRequest(index string, id string) *Request {
	for _, r := range i.requests {
		if r == nil {
			logrus.WithFields(logrus.Fields{"index": index, "id": id}).Error("Request not found")
			return nil
		}

		if r.Index == index && r.Id == id {
			return r
		}
	}

	return nil
}

func (i *Index) BatchPersist(height uint64) {
	if height%uint64(i.bulkIndexSize) != 0 || len(i.requests) == 0 {
		return
	}

	actions := i.Persist()
	logrus.WithFields(logrus.Fields{"actions": actions}).Info("Indexed height ", height)
}

func (i *Index) Persist() int {
	bulk := i.Client.Bulk()
	for _, r := range i.requests {
		if r.Type == IndexRequest {
			bulk.Add(elastic.NewBulkIndexRequest().Index(r.Index).Id(r.Id).Doc(r.Doc))
		} else if r.Type == UpdateRequest {
			bulk.Add(elastic.NewBulkUpdateRequest().Index(r.Index).Id(r.Id).Doc(r.Doc))
		}
	}

	actions := bulk.NumberOfActions()
	if actions != 0 {
		response, err := bulk.Do(context.Background())
		if err != nil {
			raven.CaptureError(err, nil)
			logrus.WithError(err).Fatal("Failed to persist requests")
		}
		if response.Errors == true {
			for _, failed := range response.Failed() {
				raven.CaptureMessage(failed.Error.Reason, nil)
				logrus.Fatal(failed.Error.Reason)
			}
		}
	}

	logrus.Debug("Persisted requests")

	i.requests = make([]*Request, 0)

	return actions
}

func (i *Index) DeleteHeightGT(height uint64, indices ...string) error {
	_, err := i.Client.DeleteByQuery(indices...).
		Query(elastic.NewRangeQuery("height").Gt(height)).
		Do(context.Background())
	if err != nil {
		raven.CaptureError(err, nil)
		logrus.WithError(err).Fatalf("Could not rewind to %d", height)
		return err
	}

	i.Client.Flush(indices...)

	logrus.Debugf("Deleted height greater than %d", height)

	return nil
}

func (i *Index) createIndex(index string, mapping []byte) error {
	ctx := context.Background()
	client := i.Client

	exists, err := client.IndexExists(index).Do(ctx)
	if err != nil {
		raven.CaptureError(err, nil)
		return err
	}

	if exists && config.Get().Reindex {
		logrus.Infof("Deleting Index: %s", index)
		_, err = client.DeleteIndex(index).Do(ctx)
		if err != nil {
			raven.CaptureError(err, nil)
			return err
		}
		exists = false
	}

	if !exists {
		createIndex, err := client.CreateIndex(index).BodyString(string(mapping)).Do(ctx)
		if err != nil {
			raven.CaptureError(err, nil)
			return err
		}

		if createIndex.Acknowledged {
			logrus.Info("Created index: ", index)
		}
	}

	return nil
}
