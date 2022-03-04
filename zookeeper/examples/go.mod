module github.com/smartwalle/xid/zookeeper/examples

go 1.12

require (
	github.com/smartwalle/xid v1.0.6
	github.com/smartwalle/xid/zookeeper v0.0.0
)

replace (
	github.com/smartwalle/xid => ../../
	github.com/smartwalle/xid/zookeeper => ../
)
