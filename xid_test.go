package xid

import (
	"testing"
	"fmt"
)

func TestNewXID(t *testing.T) {
	fmt.Println(NewXID())
	fmt.Println(NewXID())
	fmt.Println(NewXID())
	fmt.Println(NewXID())
	fmt.Println(NewXID())
}
