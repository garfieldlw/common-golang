package es

import (
	"context"
	"fmt"
	"github.com/garfieldlw/common-golang/pkg/log"
	"github.com/olivere/elastic/v7"
	"go.uber.org/zap"
	"sync"
)

type Index string

const (
	UserIndex Index = "user"
)

var lock = &sync.Mutex{}
var search *ElasticSearch

type ElasticSearch struct {
	client *elastic.Client
}

func NewElasticClient() *ElasticSearch {
	if search != nil {
		return search
	}

	lock.Lock()
	defer lock.Unlock()

	if search != nil {
		return search
	}

	esConfig := GetElasticsearchConfig()
	if esConfig == nil {
		log.Warn("get elasticsearch config failed")
		return nil
	}

	var options []elastic.ClientOptionFunc
	options = append(options, elastic.SetURL(fmt.Sprintf("https://%s:%d", esConfig.Host, esConfig.Port)))
	options = append(options, elastic.SetSniff(false))
	options = append(options, elastic.SetBasicAuth(esConfig.Username, esConfig.Password))

	client, err := elastic.NewClient(options...)
	if err != nil {
		log.Info("es new client fail", zap.Error(err))
		return nil
	}

	search = &ElasticSearch{
		client: client,
	}

	_ = search.CreateIndex(UserIndex, UserSetting)

	return search
}

func (e *ElasticSearch) CreateIndex(name Index, setting string) error {
	exists, err := e.client.IndexExists(e.getIndex(name)).Do(context.Background())
	if err != nil {
		log.Warn("es create index error", zap.Error(err))
		return err
	}
	// create index
	if !exists {
		createIndex, err := e.client.CreateIndex(e.getIndex(name)).Body(setting).Do(context.Background())
		if err != nil {
			log.Error(err.Error())
			return err
		}

		if !createIndex.Acknowledged {
			// Not acknowledged
		}
	}
	return nil
}

func (e *ElasticSearch) GetIndex(name Index) *elastic.IndexService {
	return e.client.Index().Index(e.getIndex(name))
}

func (e *ElasticSearch) AddDocument(ctx context.Context, name Index, id string, doc interface{}) error {
	_ = e.DeleteDocument(ctx, name, id)

	_, err := e.client.Index().Index(e.getIndex(name)).Id(id).BodyJson(doc).Do(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (e *ElasticSearch) DeleteDocument(ctx context.Context, name Index, id string) error {
	_, err := e.client.Delete().Index(e.getIndex(name)).Id(id).Do(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (e *ElasticSearch) UpdateByQuery(ctx context.Context, name Index, query elastic.Query, script *elastic.Script) error {
	_, err := e.client.UpdateByQuery(e.getIndex(name)).Query(query).Script(script).Refresh("true").Do(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (e *ElasticSearch) SearchAfterDocument(ctx context.Context, name Index, limit int, query elastic.Query, sort []elastic.Sorter, searchAfter []interface{}) (*elastic.SearchResult, error) {
	if limit <= 0 {
		limit = 10
	}

	res, err := e.client.Search(e.getIndex(name)).Query(query).From(0).Size(limit).SortBy(sort...).SearchAfter(searchAfter...).Do(ctx)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (e *ElasticSearch) Search(ctx context.Context, name Index, from, limit int, query elastic.Query, sort []elastic.Sorter) (*elastic.SearchResult, error) {
	if limit <= 0 {
		limit = 10
	}

	res, err := e.client.Search(e.getIndex(name)).Query(query).From(from).Size(limit).SortBy(sort...).Do(ctx)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (e *ElasticSearch) GetBulk(name Index) *elastic.BulkService {
	return e.client.Bulk().Index(e.getIndex(name))
}

func (e *ElasticSearch) getIndex(index Index) string {
	return string(index)
}

type ElasticsearchConfigItem struct {
	Host     string `json:"host"`
	Port     int32  `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func GetElasticsearchConfig() *ElasticsearchConfigItem {
	return &ElasticsearchConfigItem{
		Host:     "10.0.0.1",
		Port:     9000,
		Username: "username",
		Password: "password",
	}
}
