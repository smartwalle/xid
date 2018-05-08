package xid

import (
	"testing"
	"fmt"
	"time"
)

func TestNewXID(t *testing.T) {


	fmt.Println("秒秒（ s）", time.Now().Unix()) //获取当前秒（s）
	fmt.Println("纳秒（ns）", time.Now().UnixNano())//获取当前纳秒（ns）
	fmt.Println("微秒（µs）", time.Now().UnixNano()/1e3)//获取当前微秒（µs）
	fmt.Println("毫秒（ms）", time.Now().UnixNano()/1e6)//将纳秒转换为毫秒（ms）
	fmt.Println("秒秒（ s）", time.Now().UnixNano()/1e9)//将纳秒转换为秒（s）

	return


	fmt.Println(NewXID())
	fmt.Println(NewMID())
}

func BenchmarkNewXID(b *testing.B) {
	for i :=0; i<b.N; i++ {
		NewXID()
	}
}
