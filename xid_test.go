package xid

import (
	"fmt"
	"testing"
)

func TestXID_Next(t *testing.T) {
	for i := 0; i < 100000; i++ {
		fmt.Println(Next())
	}
}

func BenchmarkXID_Next(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Next()
	}
}
