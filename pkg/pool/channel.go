package pool

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type Config struct {
	InitialCap  int
	MaxCap      int
	MaxIdle     int
	Factory     func() (interface{}, error)
	Close       func(interface{}) error
	Ping        func(interface{}) error
	IdleTimeout time.Duration
}

type connReq struct {
	idleConn *idleConn
}

type channelPool struct {
	mu                 sync.RWMutex
	connections        chan *idleConn
	factory            func() (interface{}, error)
	close              func(interface{}) error
	ping               func(interface{}) error
	idleTimeout        time.Duration
	waitTimeOut        time.Duration
	maxActive          int
	openingConnections int
	connReqs           []chan connReq
}

type idleConn struct {
	conn interface{}
	t    time.Time
}

func NewChannelPool(poolConfig *Config) (Pool, error) {
	if !(poolConfig.InitialCap <= poolConfig.MaxIdle && poolConfig.MaxCap >= poolConfig.MaxIdle && poolConfig.InitialCap >= 0) {
		return nil, errors.New("invalid capacity settings")
	}

	if poolConfig.Factory == nil {
		return nil, errors.New("invalid factory func settings")
	}

	if poolConfig.Close == nil {
		return nil, errors.New("invalid close func settings")
	}

	c := &channelPool{
		connections:        make(chan *idleConn, poolConfig.MaxIdle),
		factory:            poolConfig.Factory,
		close:              poolConfig.Close,
		idleTimeout:        poolConfig.IdleTimeout,
		maxActive:          poolConfig.MaxCap,
		openingConnections: poolConfig.InitialCap,
	}

	if poolConfig.Ping != nil {
		c.ping = poolConfig.Ping
	}

	for i := 0; i < poolConfig.InitialCap; i++ {
		conn, err := c.factory()
		if err != nil {
			c.Release()
			return nil, fmt.Errorf("factory is not able to fill the pool: %s", err)
		}
		c.connections <- &idleConn{conn: conn, t: time.Now()}
	}

	return c, nil
}

func (c *channelPool) getConnections() chan *idleConn {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.connections
}

func (c *channelPool) Get() (interface{}, error) {
	cons := c.getConnections()
	if cons == nil {
		return nil, errors.New("pool is closed")
	}

	for {
		select {
		case wrapConn := <-cons:
			if wrapConn == nil {
				return nil, errors.New("pool is closed")
			}

			if timeout := c.idleTimeout; timeout > 0 {
				if wrapConn.t.Add(timeout).Before(time.Now()) {
					_ = c.Close(wrapConn.conn)
					continue
				}
			}

			if c.ping != nil {
				if err := c.Ping(wrapConn.conn); err != nil {
					_ = c.Close(wrapConn.conn)
					continue
				}
			}

			return wrapConn.conn, nil
		default:
			c.mu.Lock()

			if c.openingConnections >= c.maxActive {
				req := make(chan connReq, 1)
				c.connReqs = append(c.connReqs, req)

				c.mu.Unlock()

				ret, ok := <-req
				if !ok {
					return nil, errors.New("max connections")
				}

				if timeout := c.idleTimeout; timeout > 0 {
					if ret.idleConn.t.Add(timeout).Before(time.Now()) {
						_ = c.Close(ret.idleConn.conn)
						continue
					}
				}

				return ret.idleConn.conn, nil
			}

			if c.factory == nil {
				c.mu.Unlock()
				return nil, errors.New("pool is closed")
			}

			conn, err := c.factory()
			if err != nil {
				c.mu.Unlock()
				return nil, err
			}

			c.openingConnections++
			c.mu.Unlock()

			return conn, nil
		}
	}
}

func (c *channelPool) Put(conn interface{}) error {
	if conn == nil {
		return errors.New("connection is nil. rejecting")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connections == nil {
		return c.Close(conn)
	}

	if l := len(c.connReqs); l > 0 {
		req := c.connReqs[0]
		copy(c.connReqs, c.connReqs[1:])
		c.connReqs = c.connReqs[:l-1]
		req <- connReq{
			idleConn: &idleConn{conn: conn, t: time.Now()},
		}

		return nil
	} else {
		select {
		case c.connections <- &idleConn{conn: conn, t: time.Now()}:
			return nil
		default:
			return c.Close(conn)
		}
	}
}

func (c *channelPool) Close(conn interface{}) error {
	if conn == nil {
		return errors.New("connection is nil. rejecting")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.close == nil {
		return nil
	}

	c.openingConnections--
	return c.close(conn)
}

func (c *channelPool) Ping(conn interface{}) error {
	if conn == nil {
		return errors.New("connection is nil. rejecting")
	}

	return c.ping(conn)
}

func (c *channelPool) Release() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.connections = nil
	c.factory = nil
	c.ping = nil
	closeFun := c.close
	c.close = nil

	if c.connections == nil {
		return
	}

	close(c.connections)
	for wrapConn := range c.connections {
		_ = closeFun(wrapConn.conn)
	}

	c.connections = nil
}

func (c *channelPool) Len() int {
	return len(c.getConnections())
}
