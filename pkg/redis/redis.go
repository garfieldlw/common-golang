package redis

import (
	"context"
	"errors"
	"github.com/redis/go-redis/v9"
	"sync"
	"time"
)

var redisClient *Redis
var lock *sync.Mutex = &sync.Mutex{}

// Redis provides a cache backed by a Redis server.
type Redis struct {
	Config *redis.Options
	Client *redis.Client
}

// New returns an initialized Redis cache object.
func New(config *redis.Options) *Redis {
	client := redis.NewClient(config)
	return &Redis{Config: config, Client: client}
}

func (r *Redis) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.Client.TTL(ctx, key).Result()
}

func (r *Redis) Del(ctx context.Context, key string) error {
	return r.Client.Del(ctx, key).Err()
}

// Get returns the value saved under a given key.
func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	return r.Client.Get(ctx, key).Result()
}

func (r *Redis) GetBytes(ctx context.Context, key string) ([]byte, error) {
	return r.Client.Get(ctx, key).Bytes()
}

func (r *Redis) MGet(ctx context.Context, keys []string) ([]any, error) {
	return r.Client.MGet(ctx, keys...).Result()
}

// Set saves an arbitrary value under a specific key.
func (r *Redis) Set(ctx context.Context, key string, value any, expire time.Duration) error {
	return r.Client.Set(ctx, key, value, expire).Err()
}

func (r *Redis) SetNX(ctx context.Context, key string, value any, expire time.Duration) (bool, error) {
	return r.Client.SetNX(ctx, key, value, expire).Result()
}

func (r *Redis) HGet(ctx context.Context, key, field string) (string, error) {
	return r.Client.HGet(ctx, key, field).Result()
}

func (r *Redis) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return r.Client.HGetAll(ctx, key).Result()
}

func (r *Redis) HSet(ctx context.Context, key, field, value string, expire time.Duration) error {
	err := r.Client.HSet(ctx, key, field, value).Err()
	if err == nil && expire > 0 {
		r.Client.Expire(ctx, key, expire)
	}
	return err
}

func (r *Redis) HSetNX(ctx context.Context, key, field, value string, expire time.Duration) error {
	err := r.Client.HSetNX(ctx, key, field, value).Err()
	if err == nil && expire > 0 {
		r.Client.Expire(ctx, key, expire)
	}
	return err
}

func (r *Redis) HDel(ctx context.Context, key string, field ...string) error {
	return r.Client.HDel(ctx, key, field...).Err()
}

func (r *Redis) ZAdd(ctx context.Context, key string, score float64, data string) error {
	return r.Client.ZAdd(ctx, key, redis.Z{Score: score, Member: data}).Err()
}

func (r *Redis) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return r.Client.ZRange(ctx, key, start, stop).Result()
}

func (r *Redis) ZCount(ctx context.Context, key, min, max string) (int64, error) {
	return r.Client.ZCount(ctx, key, min, max).Result()
}

func (r *Redis) ZRem(ctx context.Context, key string, members ...any) error {
	return r.Client.ZRem(ctx, key, members...).Err()
}

func (r *Redis) RPush(ctx context.Context, key string, values ...any) error {
	return r.Client.RPush(ctx, key, values...).Err()
}

func (r *Redis) LPush(ctx context.Context, key string, values ...any) error {
	return r.Client.LPush(ctx, key, values...).Err()
}

func (r *Redis) LLen(ctx context.Context, key string) (int64, error) {
	return r.Client.LLen(ctx, key).Result()
}

func (r *Redis) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return r.Client.LRange(ctx, key, start, stop).Result()
}

func (r *Redis) Expire(ctx context.Context, key string, expire time.Duration) error {
	return r.Client.Expire(ctx, key, expire).Err()
}

func (r *Redis) Incr(ctx context.Context, key string) (int64, error) {
	return r.Client.Incr(ctx, key).Result()
}

func (r *Redis) Ping(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}

func GetRedis() (*Redis, error) {
	if redisClient != nil {
		return redisClient, nil
	}

	lock.Lock()
	defer lock.Unlock()

	conf := getRedisConfig()
	if conf == nil {
		return nil, errors.New("redis config is invalid")
	}

	if redisClient != nil {
		return redisClient, nil
	}

	redisClient = New(&redis.Options{
		Network:     "tcp",
		Password:    conf.Password,
		Addr:        conf.Address,
		DB:          int(conf.DB),
		DialTimeout: time.Second,
		PoolSize:    50,
		PoolTimeout: time.Second,
	})

	return redisClient, nil
}

func InitRedis() error {
	if redisClient != nil {
		return nil
	}

	lock.Lock()
	defer lock.Unlock()

	conf := getRedisConfig()
	if conf == nil {
		return errors.New("redis config is invalid")
	}

	if redisClient != nil {
		return nil
	}

	redisClient = New(&redis.Options{
		Network:     "tcp",
		Password:    conf.Password,
		Addr:        conf.Address,
		DB:          int(conf.DB),
		DialTimeout: time.Second,
		PoolSize:    50,
		PoolTimeout: time.Second,
	})

	return nil
}

func Ping(ctx context.Context) error {
	c, err := GetRedis()
	if err != nil {
		return err
	}

	err = c.Ping(ctx)
	if err != nil {
		return err
	}

	return nil
}

type ConfigItem struct {
	Address  string `json:"address"`
	Password string `json:"password"`
	DB       int32  `json:"db"`
}

func getRedisConfig() *ConfigItem {
	return &ConfigItem{
		Address:  "",
		Password: "",
		DB:       0,
	}
}
