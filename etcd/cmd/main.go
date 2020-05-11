package main

import (
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"github.com/smartwalle/xid"
	"github.com/smartwalle/xid/etcd"
)

func main() {
	etcdCli, err := clientv3.New(clientv3.Config{Endpoints: []string{"127.0.0.1:2379"}})
	if err != nil {
		return
	}

	for i := 0; i < 100; i++ {
		id, err := xid.New(etcd.WithETCD(etcdCli, "test"))
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println(id.Next(), id.DataNode())
	}

	select {}
}
