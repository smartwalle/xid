package xid_test

import (
	"fmt"
	"github.com/smartwalle/xid"
	"testing"
)

func TestNewXID(t *testing.T) {
	fmt.Println(xid.NewMID())
}

func BenchmarkNewXID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		xid.NewMID()
	}
}
