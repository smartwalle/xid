package redis

//
//import (
//	"fmt"
//	"github.com/smartwalle/dbr"
//	"github.com/smartwalle/xid"
//	"time"
//)
//
//const (
//	kTTL = 30
//)
//
//func WithDataNode(rPool dbr.Pool, key string) xid.Option {
//	var rSess = rPool.GetSession()
//
//	var node = -1
//	var nKey string
//	var maxDataNode = int(xid.MaxDataNode)
//	for i := 0; i <= maxDataNode; i++ {
//		nKey = fmt.Sprintf("xid:%s:%d", key, i)
//		var rResult = rSess.SET(nKey, i, "EX", kTTL, "NX")
//		if rResult.Error != nil {
//			return func(x *xid.XID) error {
//				rSess.Close()
//				panic(rResult.Error)
//			}
//		}
//		if rResult.MustString() == "OK" {
//			node = i
//			break
//		}
//	}
//
//	if node < 0 {
//		return func(x *xid.XID) error {
//			rSess.Close()
//			panic(xid.ErrDataNodeNotAllowed)
//		}
//	}
//
//	go func() {
//		defer rSess.Close()
//
//		var ticker = time.NewTicker(time.Second * (kTTL - 5))
//		for {
//			select {
//			case <-ticker.C:
//				rSess.EXPIRE(nKey, kTTL)
//			}
//		}
//	}()
//
//	return xid.WithDataNode(int64(node))
//}
