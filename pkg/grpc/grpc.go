package grpc

import (
	"context"
	"errors"
	"github.com/garfieldlw/common-golang/pkg/log"
	"github.com/garfieldlw/common-golang/pkg/pool"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"sync"
	"time"
)

type ServiceGrpcPoolConfig struct {
	serviceName string
	address     string
	init        int
	idle        int
	capacity    int
	idleTimeout time.Duration
}

func WithClientInterceptor() grpc.DialOption {
	return grpc.WithUnaryInterceptor(clientInterceptor())
}

func WithStreamInterceptor() grpc.DialOption {
	return grpc.WithStreamInterceptor(streamClientInterceptor())
}

func WithKeepaliveParams() grpc.DialOption {
	return grpc.WithKeepaliveParams(
		keepalive.ClientParameters{
			Time:                time.Second * 10,
			Timeout:             time.Second * 3,
			PermitWithoutStream: true,
		})
}

func NewServiceGrpcConfig(name, address string, idleTime time.Duration, init, idle, capacity int) *ServiceGrpcPoolConfig {
	return &ServiceGrpcPoolConfig{
		serviceName: name,
		address:     address,
		init:        init,
		idle:        idle,
		capacity:    capacity,
		idleTimeout: idleTime,
	}
}

func clientInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, resp interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		start := time.Now()

		err := invoker(ctx, method, req, resp, cc, opts...)
		if err != nil {
			log.Error("Invoked RPC Error[Client]", zap.String("method", method), zap.Error(err))
		}

		log.Info("Invoked RPC[Client]", zap.String("method", method), zap.String("Duration", time.Since(start).String()), zap.Error(err))
		return err
	}
}

func streamClientInterceptor() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		start := time.Now()
		cs, err := streamer(ctx, desc, cc, method, opts...)

		if err != nil {
			log.Error("Invoked RPC Error[Stream]", zap.String("method", method), zap.Error(err))
		}

		log.Info("Invoked RPC[Stream]", zap.String("method", method), zap.String("Duration", time.Since(start).String()), zap.Error(err))

		return cs, err
	}
}

func CreatePoll(config *ServiceGrpcPoolConfig, opts ...grpc.DialOption) (pool.Pool, error) {
	if opts == nil {
		opts = []grpc.DialOption{}
	}

	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()), WithClientInterceptor(), WithStreamInterceptor(), WithKeepaliveParams())

	return pool.NewChannelPool(&pool.Config{
		Factory: func() (interface{}, error) {
			return grpc.Dial(config.address, opts...)
		},
		InitialCap: config.init,
		MaxIdle:    config.idle,
		MaxCap:     config.capacity,
		Close: func(i interface{}) error {
			if v, ok := i.(*grpc.ClientConn); ok {
				return v.Close()
			}
			return nil
		},

		IdleTimeout: config.idleTimeout,
		//Ping: func(i interface{}) error {
		//	if v, ok := i.(*grpc.ClientConn); ok {
		//		if v.GetState() == connectivity.Connecting || v.GetState() == connectivity.Ready || v.GetState() == connectivity.Idle {
		//			return nil
		//		}
		//	}
		//	return errors.New("connect closed")
		//},
	})
}

var (
	grpcServiceMap = make(map[string]pool.Pool)
	lock           = &sync.Mutex{}
)

func LoadServicePool(serviceName string) (pool.Pool, error) {
	if v, ok := grpcServiceMap[serviceName]; ok {
		if p, valid := v.(pool.Pool); valid {
			return p, nil
		} else {
			delete(grpcServiceMap, serviceName)
		}
	}

	lock.Lock()
	defer lock.Unlock()

	if v, ok := grpcServiceMap[serviceName]; ok {
		if p, valid := v.(pool.Pool); valid {
			return p, nil
		} else {
			delete(grpcServiceMap, serviceName)
		}
	}

	value := GetGrpcHostByServiceName(serviceName)
	if value == nil {
		log.Warn("get grpc config error")
		return nil, errors.New("get grpc config error")
	}

	p, err := CreatePoll(NewServiceGrpcConfig(serviceName, value.Address, time.Second, value.Init, value.Idle, value.Capacity))
	if err != nil {
		log.Warn("get pool error", zap.Error(err))
		return nil, err
	}

	grpcServiceMap[serviceName] = p

	return grpcServiceMap[serviceName], nil
}

type ConfigItem struct {
	Name     string `json:"name"`
	Address  string `json:"address"`
	Init     int    `json:"init"`
	Idle     int    `json:"idle"`
	Capacity int    `json:"capacity"`
}

var allGrpc = make(map[string]*ConfigItem)
var once sync.Once

func init() {
	once.Do(
		func() {
			allGrpc["user"] = &ConfigItem{
				Name:     "user",
				Address:  "host:port",
				Init:     5,
				Idle:     5,
				Capacity: 5,
			}
		},
	)
}

func GetGrpcHostByServiceName(name string) *ConfigItem {
	if allGrpc == nil || len(allGrpc) == 0 {
		return nil
	}

	if item, ok := allGrpc[name]; ok {
		return item
	}

	return nil
}
