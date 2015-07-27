package elastic

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/olivere/elastic"
)

const (
	pageLimit = 20
)

// ElasticSearch .
type ElasticSearch struct {
	uri    string
	index  string
	client *elastic.Client
}

// NewElasticSearchSession .
func New(uri, index string) *ElasticSearch {
	return &ElasticSearch{uri, index, nil}
}

func (es *ElasticSearch) Conn() error {
	client, err := elastic.NewClient(
		elastic.SetURL(es.uri),
		elastic.SetSniff(false),
		elastic.SetHealthcheckInterval(10*time.Second),
		elastic.SetMaxRetries(5))

	if err != nil {
		// Handle error
		return err
	}

	exists, err := client.IndexExists(es.index).Do()
	if err != nil {
		// Handle error
		return err
	}
	if !exists {
		// Create a new index.
		createIndex, err := client.CreateIndex(es.index).Do()
		if err != nil {
			// Handle error
			return err
		}
		if !createIndex.Acknowledged {
			// Not acknowledged
		}
	}

	es.client = client
	return nil
}

func (es *ElasticSearch) Find(query elastic.Query, table string, params ...int) ([]interface{}, int64, error) {
	var objects []interface{}

	skipCount := 0

	if len(params) >= 1 {
		if params[0] > 1 {
			skipCount = (params[0] - 1) * pageLimit
		}
	}

	searchResult, err := es.client.Search().
		Index(es.index).
		Type(table).
		Query(query).
		From(skipCount).Size(pageLimit).
		Pretty(true).
		Do()
	if err != nil {
		return nil, 0, err
	}

	fmt.Printf("Query took %d milliseconds\n", searchResult.TookInMillis)

	if searchResult.Hits != nil {
		// Iterate through results
		for _, hit := range searchResult.Hits.Hits {
			var model interface{}

			err := json.Unmarshal(*hit.Source, &model)
			if err != nil {
				return nil, 0, err
			}

			objects = append(objects, model)

		}
	}

	return objects, searchResult.Hits.TotalHits, nil
}

func (es *ElasticSearch) DeleteIndex() {
	es.client.DeleteIndex(es.index).Do()
}

func (es *ElasticSearch) Insert(table string, model interface{}) error {
	_, err := es.client.Index().
		Index(es.index).
		Type(table).
		BodyJson(model).
		Do()

	if err != nil {
		return err
	}

	// Flush data
	_, err = es.client.Flush().Index(es.index).Do()

	if err != nil {
		return err
	}

	return nil
}

func (es *ElasticSearch) Delete(table string, query elastic.Query) error {
	_, err := es.client.DeleteByQuery().Index(es.index).Type(table).Query(query).Do()
	if err != nil {
		return err
	}

	_, err = es.client.Flush().Index(es.index).Do()
	if err != nil {
		return err
	}
	return nil
}

//bulk methods

func (es *ElasticSearch) NewBulk() *elastic.BulkService {
	return es.client.Bulk()
}

func (es *ElasticSearch) AddToBulk(bulk *elastic.BulkService, table string, model interface{}) {
	bulk.Add(elastic.NewBulkIndexRequest().Index(es.index).Type(table).Doc(model))
}

func (es *ElasticSearch) SendBulk(bulk *elastic.BulkService) {
	bulk.Do()
}
