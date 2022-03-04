package main

import (
	"fmt"
	"github.com/smartwalle/xid"
	"github.com/smartwalle/xid/etcd"
	"go.etcd.io/etcd/client/v3"
)

func main() {
	etcdCli, err := clientv3.New(clientv3.Config{Endpoints: []string{"127.0.0.1:2379"}})
	if err != nil {
		return
	}

	for i := 0; i < 100; i++ {
		id, err := xid.New(etcd.WithDataNode(etcdCli, "test"))
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println(id.Next(), id.DataNode())
	}

	select {}
}
