module example.com/gogo-modern-integration

go 1.23

require (
	github.com/cockroachdb/errors v1.8.1
	github.com/gogo/protobuf v1.3.2
	go.etcd.io/etcd/raft/v3 v3.5.0
	google.golang.org/protobuf v1.36.13-0.20261007182434-0c373ff94f29
)

require (
	github.com/cockroachdb/logtags v0.0.0-20190617123548-eb05cc24525f // indirect
	github.com/cockroachdb/redact v1.0.8 // indirect
	github.com/cockroachdb/sentry-go v0.6.1-cockroachdb.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/kr/pretty v0.1.0 // indirect
	github.com/kr/text v0.1.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
)

replace github.com/gogo/protobuf => ../..
