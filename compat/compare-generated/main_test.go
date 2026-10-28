package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestComparisonPreservesCodeAndDescriptorChecks(t *testing.T) {
	dir := t.TempDir()
	makeFile := func(name, comment, tag, body string, raw []byte, level int) []byte {
		var compressed bytes.Buffer
		z, _ := gzip.NewWriterLevel(&compressed, level)
		z.Write(raw)
		z.Close()
		source := fmt.Sprintf("package fixture\n%s\ntype M struct { X int32 `%s` }\nfunc F() int {return %s}\nvar fileDescriptor_abc = []byte{%s}\n", comment, tag, body, byteLiterals(compressed.Bytes()))
		path := filepath.Join(dir, name)
		if err := ioutil.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
		result, err := normalized(path)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	original := makeFile("a.go", "// old comment", "protobuf:\"varint,1,opt,name=x\"", "1", []byte{1, 2}, gzip.BestCompression)
	same := makeFile("b.go", "// different comment\n//\n// layout", "protobuf:\"varint,1,opt,name=x\"", "1", []byte{1, 2}, gzip.NoCompression)
	if !bytes.Equal(original, same) {
		t.Fatal("representation-only difference rejected")
	}
	for _, tc := range []struct {
		tag, body string
		raw       []byte
	}{
		{"protobuf:\"varint,2,opt,name=x\"", "1", []byte{1, 2}},
		{"protobuf:\"varint,1,opt,name=x\"", "2", []byte{1, 2}},
		{"protobuf:\"varint,1,opt,name=x\"", "1", []byte{1, 3}},
	} {
		if bytes.Equal(original, makeFile("c.go", "", tc.tag, tc.body, tc.raw, gzip.BestCompression)) {
			t.Fatal("meaningful difference accepted")
		}
	}
}
func byteLiterals(b []byte) string {
	s := ""
	for _, n := range b {
		s += fmt.Sprintf("%d,", n)
	}
	return s
}
