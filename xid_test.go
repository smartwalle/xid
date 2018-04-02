package xid

import (
	"testing"
	"fmt"
)

func TestNewXID(t *testing.T) {
	fmt.Println(NewXID())
	fmt.Println(NewMID())
}

func BenchmarkNewXID(b *testing.B) {
	for i :=0; i<b.N; i++ {
		NewXID()
	}
}
