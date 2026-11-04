package gogoreflect

import (
	"bytes"
	v2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"testing"
)

func metadataSet(t *testing.T, files ...*descriptorpb.FileDescriptorProto) []byte {
	t.Helper()
	b, e := v2.Marshal(&descriptorpb.FileDescriptorSet{File: files})
	if e != nil {
		t.Fatal(e)
	}
	return compressMetadata(b)
}
func TestSupplementalMetadataTransaction(t *testing.T) {
	a := &descriptorpb.FileDescriptorProto{Name: v2.String("metadata-test-a.proto"), Package: v2.String("meta.a"), Syntax: v2.String("proto3")}
	if e := RegisterFileSet(metadataSet(t, a)); e != nil {
		t.Fatal(e)
	}
	raw := metadataSet(t, a)
	if e := RegisterFileSet(raw); e != nil {
		t.Fatal("identical repeated registration", e)
	}
	b := v2.Clone(a).(*descriptorpb.FileDescriptorProto)
	b.Name = v2.String("metadata-test-b.proto")
	conflict := v2.Clone(a).(*descriptorpb.FileDescriptorProto)
	conflict.Package = v2.String("meta.conflict")
	if e := RegisterFileSet(metadataSet(t, b, conflict)); e == nil {
		t.Fatal("conflicting batch accepted")
	}
	schemas.Lock()
	defer schemas.Unlock()
	if schemas.supplemental[b.GetName()] != nil {
		t.Fatal("partial transaction was published")
	}
	got, e := inflate(schemas.source(a.GetName()))
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(got, schemas.supplemental[a.GetName()]) {
		t.Fatal("supplemental source changed")
	}
}
func TestSupplementalMetadataRejectsCorruptInput(t *testing.T) {
	for _, raw := range [][]byte{nil, []byte("bad gzip"), compressMetadata([]byte{0xff})} {
		if e := RegisterFileSet(raw); e == nil {
			t.Fatal("corrupt metadata accepted")
		}
	}
}
