package xid

import (
	"errors"
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"path"
)

var ErrInvalidZKConnection = errors.New("xid: invalid zookeeper connection")

func WithZookeeper(conn *zk.Conn, key string) Option {
	if conn == nil {
		return func(x *XID) error {
			return ErrInvalidZKConnection
		}
	}

	var lockPath = path.Join(fmt.Sprintf("/smartwalle/xid/%s/locker", key))
	var lock = zk.NewLock(conn, lockPath, zk.WorldACL(zk.PermAll))
	if err := lock.Lock(); err != nil {
		lock.Unlock()
		return func(x *XID) error {
			return err
		}
	}

	children, _, err := conn.Children(fmt.Sprintf("/smartwalle/xid/%s", key))
	if err != nil {
		lock.Unlock()
		return func(x *XID) error {
			return err
		}
	}
	var existsNode = make(map[string]struct{}, len(children))
	for _, child := range children {
		existsNode[child] = struct{}{}
	}

	var nNode = -1
	for i := 0; i <= int(kMaxDataNode); i++ {
		if _, exists := existsNode[fmt.Sprintf("node-%d", i)]; exists {
			continue
		}

		var nPath = path.Join(fmt.Sprintf("/smartwalle/xid/%s/node-%d", key, i))
		exists, _, err := conn.Exists(nPath)
		if err != nil {
			lock.Unlock()
			return func(x *XID) error {
				return err
			}
		}
		if exists == true {
			continue
		}

		cPath, err := conn.Create(nPath, []byte{}, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
		if err != nil {
			lock.Unlock()
			return func(x *XID) error {
				return err
			}
		}

		if cPath != nPath {
			continue
		}

		nNode = i
		break
	}

	if err := lock.Unlock(); err != nil {
		return func(x *XID) error {
			return err
		}
	}

	return WithDataNode(int64(nNode))
}
