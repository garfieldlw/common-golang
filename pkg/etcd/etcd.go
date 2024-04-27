package etcd

import (
	"context"
	"errors"
	client3 "go.etcd.io/etcd/client/v3"
	"sync"
	"time"
)

const TIMEOUT = 1000 * time.Millisecond

var lock = &sync.Mutex{}
var etcdInstance *Etcd

type conf struct {
	Urls     []string
	UserName string
	Password string
}

var etcdConf = &conf{
	Urls:     []string{},
	UserName: "",
	Password: "",
}

type Etcd struct {
	cli *client3.Client
}

type Item struct {
	Path  string `json:"path"`
	Value string `json:"value"`
}

func Service() *Etcd {
	if etcdInstance != nil {
		return etcdInstance
	}

	lock.Lock()
	defer lock.Unlock()

	if etcdInstance != nil {
		return etcdInstance
	}

	etcdInstance = initEtcd(etcdConf)

	return etcdInstance
}

func initEtcd(etcdConfig *conf) *Etcd {
	cli, err := client3.New(client3.Config{
		DialTimeout: TIMEOUT,
		Endpoints:   etcdConfig.Urls,
		Username:    etcdConfig.UserName,
		Password:    etcdConfig.Password,
	})
	if err != nil {
		panic("connect etcd failed")
	}
	return &Etcd{cli: cli}
}

func (e *Etcd) Delete(ctx context.Context, keyPath string) error {
	kv := client3.KV(e.cli)
	ctx, _ = context.WithTimeout(ctx, TIMEOUT)
	_, err := kv.Delete(ctx, keyPath)
	if err != nil {
		return err
	}
	return nil
}

func (e *Etcd) Put(ctx context.Context, keyPath, value string) error {
	kv := client3.KV(e.cli)
	ctx, _ = context.WithTimeout(ctx, TIMEOUT)
	_, err := kv.Put(ctx, keyPath, value)
	if err != nil {
		return err
	}
	return nil
}

func (e *Etcd) Get(ctx context.Context, keyPath string) (string, error) {
	kv := client3.KV(e.cli)
	ctx, _ = context.WithTimeout(ctx, TIMEOUT)
	res, err := kv.Get(ctx, keyPath)
	if err != nil {
		return "", err
	}

	for _, val := range res.Kvs {
		if string(val.Key[:]) == keyPath {
			return string(val.Value[:]), nil
		}
	}

	return "", errors.New("no value in etcd")
}

func (e *Etcd) GetBytes(ctx context.Context, keyPath string) ([]byte, error) {
	kv := client3.KV(e.cli)
	ctx, _ = context.WithTimeout(ctx, TIMEOUT)
	res, err := kv.Get(ctx, keyPath)
	if err != nil {
		return nil, err
	}

	for _, val := range res.Kvs {
		if string(val.Key[:]) == keyPath {
			return val.Value, nil
		}
	}

	return nil, errors.New("no value in etcd")
}

func (e *Etcd) GetList(ctx context.Context, keyPath string) ([]*Item, error) {
	kv := client3.KV(e.cli)
	ctx, _ = context.WithTimeout(ctx, TIMEOUT)
	res, err := kv.Get(ctx, keyPath, client3.WithPrefix())
	if err != nil {
		return nil, err
	}

	var items []*Item

	for _, val := range res.Kvs {
		item := new(Item)
		item.Path = string(val.Key[:])
		item.Value = string(val.Value[:])

		items = append(items, item)
	}

	return items, nil
}

func (e *Etcd) GetClient() *client3.Client {
	return e.cli
}
