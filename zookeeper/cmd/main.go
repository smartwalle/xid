package main

import (
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/smartwalle/xid"
	"github.com/smartwalle/xid/zookeeper"
	"time"
)

func main() {
	conn, _, err := zk.Connect([]string{"127.0.0.1"}, time.Second*10)
	if err != nil {
		return
	}

	for i := 0; i < 100; i++ {
		id, err := xid.New(zookeeper.WithDataNode(conn, "test"))
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println(id.Next(), id.DataNode())
	}

	select {}
}
