package redis

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/smartwalle/xid"
)

const (
	kPrefix      = "xid"
	kMinLeaseTTL = 10 * time.Second
)

var (
	ErrInvalidKey      = errors.New("invalid redis key")
	ErrInvalidLeaseTTL = errors.New("redis lease ttl must be greater than or equal to 10s")
	ErrNoAvailableNode = errors.New("no available data node in redis")
	ErrLeaseLost       = errors.New("redis data node lease lost")
)

const renewScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("PEXPIRE", KEYS[1], ARGV[2])
end
return 0
`

const releaseScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`

type DataNode struct {
	client goredis.UniversalClient
	key    string
	token  string
	node   int64
	ttl    time.Duration

	stop      chan struct{}
	done      chan struct{}
	closeOnce sync.Once
}

func NewDataNode(client goredis.UniversalClient, key string, ttl time.Duration) (*DataNode, error) {
	if key == "" {
		return nil, ErrInvalidKey
	}

	if ttl < kMinLeaseTTL {
		return nil, ErrInvalidLeaseTTL
	}

	var token = uuid.New().String()

	var node int64
	var leaseKey string
	node, leaseKey, err := acquireDataNode(client, key, token, ttl)
	if err != nil {
		return nil, err
	}

	var dataNode = &DataNode{
		client: client,
		key:    leaseKey,
		token:  token,
		node:   node,
		ttl:    ttl,
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
	go dataNode.keepAlive()

	return dataNode, nil
}

func (node *DataNode) Node() int64 {
	return node.node
}

func (node *DataNode) Option() xid.Option {
	return xid.WithDataNode(node.node)
}

func (node *DataNode) Close() error {
	var err error
	node.closeOnce.Do(func() {
		close(node.stop)
		<-node.done
		err = node.release()
	})
	return err
}

func acquireDataNode(client goredis.UniversalClient, key string, token string, ttl time.Duration) (int64, string, error) {
	for node := int64(0); node <= int64(xid.MaxDataNode); node++ {
		var leaseKey = dataNodeKey(key, node)
		var ok, err = client.SetNX(context.Background(), leaseKey, token, ttl).Result()
		if err != nil {
			return 0, "", err
		}
		if ok {
			return node, leaseKey, nil
		}
	}
	return 0, "", ErrNoAvailableNode
}

func (node *DataNode) keepAlive() {
	defer close(node.done)

	var interval = node.ttl / 3
	if interval < time.Second {
		interval = time.Second
	}

	var ticker = time.NewTicker(interval)
	defer ticker.Stop()

	var lastRenewed = time.Now()
	for {
		select {
		case <-node.stop:
			return
		case <-ticker.C:
			var renewed, err = node.client.Eval(context.Background(), renewScript, []string{node.key}, node.token, int64(node.ttl/time.Millisecond)).Int()
			if err == nil && renewed == 1 {
				lastRenewed = time.Now()
				continue
			}
			if err == nil || time.Since(lastRenewed) >= node.ttl {
				panic(ErrLeaseLost)
			}
		}
	}
}

func (node *DataNode) release() error {
	var _, err = node.client.Eval(context.Background(), releaseScript, []string{node.key}, node.token).Int()
	return err
}

func dataNodeKey(key string, node int64) string {
	return fmt.Sprintf("%s:%s:%d", kPrefix, key, node)
}
