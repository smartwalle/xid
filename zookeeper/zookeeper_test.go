package zookeeper

import (
	"github.com/samuel/go-zookeeper/zk"
	"github.com/smartwalle/xid"
	"testing"
	"time"
)

func TestWithZookeeper(t *testing.T) {
	conn, _, err := zk.Connect([]string{"127.0.0.1"}, time.Second*10)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 100; i++ {
		id, err := xid.New(WithZookeeper(conn, "test"))
		if err != nil {
			t.Fatal(err)
		}

		t.Log(id.Next(), id.DataNode())
	}
}
