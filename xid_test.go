package xid

import (
	"fmt"
	"testing"
	"time"
)

func TestXID_Next(t *testing.T) {
	for i := 0; i < 100000; i++ {
		fmt.Println(Next())
	}
}

func TestXID_Info(t *testing.T) {
	var timeOffset = time.Date(2020, time.January, 1, 1, 1, 1, 1, time.UTC)

	var x, _ = New(WithTimeOffset(timeOffset), WithDataNode(1))

	for i := 0; i < 100; i++ {
		var id = x.Next()

		var createdOn = timeOffset.Add(time.Duration(Time(id)) * time.Second)
		var sequence = Sequence(id)
		var node = DataNode(id)

		t.Log(id, "- 生成时间:", createdOn, "数据节点:", node, "序列号:", sequence)
	}
}

func BenchmarkXID_Next(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Next()
	}
}
