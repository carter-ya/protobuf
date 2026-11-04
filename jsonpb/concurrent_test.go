package jsonpb_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/gogo/protobuf/jsonpb"
	jsonstandard "github.com/gogo/protobuf/jsonpb/standard"
	"github.com/gogo/protobuf/proto"
	descriptor "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	textstandard "github.com/gogo/protobuf/textpb/standard"
)

// Binary sizing writes internal caches even when application fields are read
// only. JSON type checks must not copy those internal caches via Interface.
func TestConcurrentReadOnlyEncoders(t *testing.T) {
	message := &descriptor.FileDescriptorProto{Options: &descriptor.FileOptions{GoPackage: proto.String("example.com/read-only")}}
	const want = `{"options":{"goPackage":"example.com/read-only"}}`
	workers := []func() error{
		func() error { _, err := proto.Marshal(message); return err },
		func() error {
			got, err := (&jsonpb.Marshaler{}).MarshalToString(message)
			if err == nil && got != want {
				return fmt.Errorf("legacy JSON: %s", got)
			}
			return err
		},
		func() error {
			got, err := (&jsonstandard.Marshaler{}).MarshalToString(message)
			if err == nil && got != want {
				return fmt.Errorf("standard JSON: %s", got)
			}
			return err
		},
		func() error { _ = proto.MarshalTextString(message); return nil },
		func() error { _, err := (&textstandard.Marshaler{}).MarshalToString(message); return err },
	}
	var wait sync.WaitGroup
	errors := make(chan error, len(workers))
	start := make(chan struct{})
	for _, encode := range workers {
		wait.Add(1)
		go func(encode func() error) {
			defer wait.Done()
			<-start
			for i := 0; i < 128; i++ {
				if err := encode(); err != nil {
					errors <- err
					return
				}
			}
		}(encode)
	}
	close(start)
	wait.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
}
