module github.com/smartwalle/xid/etcd/examples

go 1.12

require (
	github.com/smartwalle/xid v1.0.6
	github.com/smartwalle/xid/etcd v0.0.0
	go.etcd.io/etcd/client/v3 v3.5.0-alpha.0 // indirect
)

replace (
	github.com/smartwalle/xid => ../../
	github.com/smartwalle/xid/etcd => ../
)
