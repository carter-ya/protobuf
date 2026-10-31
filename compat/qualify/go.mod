module example.com/gogo-qualification

go 1.23

require (
	github.com/gogo/protobuf v1.3.2
	google.golang.org/protobuf v1.36.12
)

replace	github.com/gogo/protobuf => ../..
