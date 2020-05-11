package zookeeper

import (
	"errors"
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/smartwalle/xid"
	"path"
)

const zkPrefix = "/xid/zk/"

var ErrInvalidZKConnection = errors.New("xid: invalid zookeeper connection")

func WithDataNode(conn *zk.Conn, key string) xid.Option {
	if conn == nil {
		return func(x *xid.XID) error {
			return ErrInvalidZKConnection
		}
	}

	var lockPath = path.Join(zkPrefix, key, "/locker")
	var lock = zk.NewLock(conn, lockPath, zk.WorldACL(zk.PermAll))
	if err := lock.Lock(); err != nil {
		lock.Unlock()
		return func(x *xid.XID) error {
			return err
		}
	}
	children, _, err := conn.Children(path.Join(zkPrefix, key))
	if err != nil {
		lock.Unlock()
		return func(x *xid.XID) error {
			return err
		}
	}
	var existsNode = make(map[string]struct{}, len(children))
	for _, child := range children {
		existsNode[child] = struct{}{}
	}

	var nNode = -1
	for i := 0; i <= int(xid.MaxDataNode); i++ {
		var nValue = fmt.Sprintf("node-%d", i)
		if _, exists := existsNode[nValue]; exists {
			continue
		}

		var nPath = path.Join(zkPrefix, key, nValue)
		exists, _, err := conn.Exists(nPath)
		if err != nil {
			lock.Unlock()
			return func(x *xid.XID) error {
				return err
			}
		}
		if exists == true {
			continue
		}

		cPath, err := conn.Create(nPath, []byte{}, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
		if err != nil {
			lock.Unlock()
			return func(x *xid.XID) error {
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
		return func(x *xid.XID) error {
			return err
		}
	}

	return xid.WithDataNode(int64(nNode))
}
