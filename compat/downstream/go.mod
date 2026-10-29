module example.com/gogo-downstream-compatibility

go 1.16

require (
	github.com/cockroachdb/errors v1.8.1
	github.com/gogo/protobuf v1.3.2
	go.etcd.io/etcd/raft/v3 v3.5.0
	google.golang.org/grpc v1.29.1
)

replace github.com/gogo/protobuf => ../..
