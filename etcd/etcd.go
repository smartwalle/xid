package etcd

import (
	"context"
	"errors"
	"fmt"
	"github.com/smartwalle/xid"
	"go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"path"
)

const (
	kPrefix = "xid"
)

var ErrInvalidETCDClient = errors.New("xid: invalid ETCD client")

func WithDataNode(client *clientv3.Client, key string, opts ...concurrency.SessionOption) xid.Option {
	if client == nil {
		return func(x *xid.XID) error {
			return ErrInvalidETCDClient
		}
	}

	var session, err = concurrency.NewSession(client, opts...)
	if err != nil {
		if session != nil {
			session.Close()
		}
		return func(x *xid.XID) error {
			return ErrInvalidETCDClient
		}
	}
	defer session.Close()

	var lockPath = path.Join(kPrefix, key, "locker")
	var lock = concurrency.NewMutex(session, lockPath)
	if err = lock.Lock(context.Background()); err != nil {
		return func(x *xid.XID) error {
			return err
		}
	}
	defer lock.Unlock(context.Background())

	var kv = clientv3.NewKV(client)
	rsp, err := kv.Get(context.Background(), path.Join(kPrefix, key), clientv3.WithPrefix())
	if err != nil {
		return func(x *xid.XID) error {
			return err
		}
	}

	var existsNode = make(map[string]struct{}, len(rsp.Kvs))
	for _, kvs := range rsp.Kvs {
		existsNode[string(kvs.Value)] = struct{}{}
	}

	var nNode = -1
	for i := 0; i < int(xid.MaxDataNode); i++ {
		if _, exists := existsNode[fmt.Sprintf("node-%d", i)]; exists {
			continue
		}
		nNode = i
		break
	}

	if nNode < 0 || nNode > int(xid.MaxDataNode) {
		return func(x *xid.XID) error {
			return xid.ErrDataNodeNotAllowed
		}
	}

	var lease = clientv3.NewLease(client)
	grantRsp, err := lease.Grant(context.Background(), 120)
	if err != nil {
		return func(x *xid.XID) error {
			return err
		}
	}
	liveRsp, err := lease.KeepAlive(context.Background(), grantRsp.ID)
	if err != nil {
		return func(x *xid.XID) error {
			return err
		}
	}

	var nValue = fmt.Sprintf("node-%d", nNode)
	if _, err = kv.Put(context.Background(), path.Join(kPrefix, key, nValue), nValue, clientv3.WithLease(grantRsp.ID)); err != nil {
		lease.Revoke(context.Background(), grantRsp.ID)
		return func(x *xid.XID) error {
			return err
		}
	}

	go func() {
		for {
			select {
			case _, ok := <-liveRsp:
				if ok == false {
					lease.Revoke(context.Background(), grantRsp.ID)
					return
				}
			}
		}
	}()

	return xid.WithDataNode(int64(nNode))
}
