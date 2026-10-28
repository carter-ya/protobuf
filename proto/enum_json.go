package proto

import "sync"

type enumJSONInfo struct {
	names  map[int32]string
	values map[string]int32
}

var enumJSONRegistry = struct {
	sync.RWMutex
	types map[string]enumJSONInfo
}{types: make(map[string]enumJSONInfo)}

// RegisterEnumJSONNames is called by new generated code. Text-format names and
// the legacy enum registry remain unchanged. The first alias is canonical.
func RegisterEnumJSONNames(typeName string, names map[int32]string, values map[string]int32) {
	n := make(map[int32]string, len(names))
	for k, v := range names {
		n[k] = v
	}
	v := make(map[string]int32, len(values))
	for k, n := range values {
		v[k] = n
	}
	enumJSONRegistry.Lock()
	defer enumJSONRegistry.Unlock()
	if _, exists := enumJSONRegistry.types[typeName]; exists {
		panic("proto: duplicate enum JSON names: " + typeName)
	}
	enumJSONRegistry.types[typeName] = enumJSONInfo{n, v}
}

// EnumJSONName returns the canonical JSON name for a newly generated enum.
func EnumJSONName(typeName string, number int32) (string, bool) {
	enumJSONRegistry.RLock()
	defer enumJSONRegistry.RUnlock()
	s, ok := enumJSONRegistry.types[typeName].names[number]
	return s, ok
}

// EnumJSONValue accepts additional JSON spellings without changing text names.
func EnumJSONValue(typeName, name string) (int32, bool) {
	enumJSONRegistry.RLock()
	defer enumJSONRegistry.RUnlock()
	n, ok := enumJSONRegistry.types[typeName].values[name]
	return n, ok
}
