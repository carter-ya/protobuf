package gogoreflect

import (
	"bytes"
	"compress/gzip"
	"fmt"

	v2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// RegisterFileSet installs supplemental descriptor metadata for option-only
// dependencies, which need no generated Go import. Input is a gzipped
// FileDescriptorSet emitted by the selected generator. Native runtime
// descriptors remain authoritative when a Go binding exists. No V2 global
// registration is performed. Conflicts are rejected before publishing any data.
func RegisterFileSet(raw []byte) error {
	data, err := inflate(raw)
	if err != nil {
		return err
	}
	set := new(descriptorpb.FileDescriptorSet)
	if err = v2.Unmarshal(data, set); err != nil {
		return err
	}
	pending := make(map[string][]byte)
	for _, file := range set.File {
		file.SourceCodeInfo = nil
		file.Name = v2.String(canonicalPath(file.GetName()))
		for i, p := range file.Dependency {
			file.Dependency[i] = canonicalPath(p)
		}
		for i, p := range file.OptionDependency {
			file.OptionDependency[i] = canonicalPath(p)
		}
		b, err := (v2.MarshalOptions{Deterministic: true}).Marshal(file)
		if err != nil {
			return err
		}
		if old, ok := pending[file.GetName()]; ok && !bytes.Equal(old, b) {
			return fmt.Errorf("gogoreflect: conflicting supplemental descriptor %q", file.GetName())
		}
		pending[file.GetName()] = b
	}
	schemas.Lock()
	defer schemas.Unlock()
	for name, b := range pending {
		if old := schemas.supplemental[name]; old != nil && !bytes.Equal(old, b) {
			return fmt.Errorf("gogoreflect: conflicting supplemental descriptor %q", name)
		}
	}
	if schemas.supplemental == nil {
		schemas.supplemental = make(map[string][]byte)
	}
	for name, b := range pending {
		schemas.supplemental[name] = b
	}
	return nil
}
func compressMetadata(data []byte) []byte {
	var b bytes.Buffer
	z := gzip.NewWriter(&b)
	if _, e := z.Write(data); e != nil {
		panic(e)
	}
	if e := z.Close(); e != nil {
		panic(e)
	}
	return b.Bytes()
}
