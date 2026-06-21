package xid_test

import (
	"testing"
	"time"

	"github.com/smartwalle/xid"
)

func TestXID_Info(t *testing.T) {
	var timeOffset = time.Date(2020, time.January, 1, 1, 1, 1, 1, time.UTC)

	var x, _ = xid.New(xid.WithTimeOffset(timeOffset), xid.WithDataNode(1))

	for i := 0; i < 100; i++ {
		var id, err = x.Next()
		if err != nil {
			t.Fatal(err)
		}

		var createdOn = timeOffset.Add(time.Duration(xid.Time(id)) * time.Second)
		var sequence = xid.Sequence(id)
		var node = xid.DataNode(id)

		t.Log(id, "- 生成时间:", createdOn, "数据节点:", node, "序列号:", sequence)
	}
}

func TestWithTimeOffset_RejectsFutureTime(t *testing.T) {
	_, err := xid.New(xid.WithTimeOffset(time.Now().Add(2 * time.Second)))
	if err != xid.ErrTimeOffsetNotAllowed {
		t.Fatalf("expected %v, got %v", xid.ErrTimeOffsetNotAllowed, err)
	}
}

func TestWithTimeOffset_RejectsOverflowTime(t *testing.T) {
	const maxTime int64 = 1<<33 - 1

	var offset = time.Unix(time.Now().Unix()-maxTime-1, 0)
	_, err := xid.New(xid.WithTimeOffset(offset))
	if err != xid.ErrTimeOffsetNotAllowed {
		t.Fatalf("expected %v, got %v", xid.ErrTimeOffsetNotAllowed, err)
	}
}

func TestXID_Next(t *testing.T) {
	var x, err = xid.New()
	if err != nil {
		t.Fatal(err)
	}

	var id uint64
	id, err = x.Next()
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 {
		t.Fatal("unexpected zero id")
	}
}

func TestXID_MustNext(t *testing.T) {
	var x, err = xid.New()
	if err != nil {
		t.Fatal(err)
	}

	var id = x.MustNext()
	if id == 0 {
		t.Fatal("unexpected zero id")
	}
}

func TestMustNext(t *testing.T) {
	var id = xid.MustNext()
	if id == 0 {
		t.Fatal("unexpected zero id")
	}
}

func BenchmarkXID_Next(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := xid.Next(); err != nil {
			b.Fatal(err)
		}
	}
}
