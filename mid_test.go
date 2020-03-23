package xid

import (
	"fmt"
	"testing"
)

func TestNewXID(t *testing.T) {
	fmt.Println(NewMID())
}

func BenchmarkNewXID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewMID()
	}
}
