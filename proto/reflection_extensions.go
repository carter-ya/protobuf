package proto

import "fmt"

// UnregisteredExtensionWire returns an owned, encoded snapshot of unregistered
// map-backed extension fields. Logical reads are synchronized with
// lazy extension decoding. As with all proto APIs, callers synchronize writes.
// Byte-backed extensions are unsupported by this bridge.
func UnregisteredExtensionWire(pb Message) (map[int32][]byte, error) {
	if _, ok := pb.(extensionsBytes); ok {
		return nil, fmt.Errorf("proto: reflection requires map-backed extensions")
	}
	epb, err := extendable(pb)
	if err != nil {
		return nil, err
	}
	fields, mu := epb.extensionsRead()
	if fields == nil {
		return nil, nil
	}
	mu.Lock()
	defer mu.Unlock()
	registered := RegisteredExtensions(pb)
	out := make(map[int32][]byte, len(fields))
	for n, e := range fields {
		if registered[n] != nil {
			continue
		}
		if err := e.Encode(); err != nil {
			return nil, err
		}
		out[n] = append([]byte(nil), e.enc...)
	}
	return out, nil
}

// ReplaceUnregisteredExtensions replaces only unregistered map-backed extension
// fields, preserving every registered field. Values must contain valid wire
// occurrences of their key field number; the reflection bridge validates this
// before calling. Input byte slices are copied. Callers synchronize writes.
func ReplaceUnregisteredExtensions(pb Message, fields map[int32][]byte) error {
	if _, ok := pb.(extensionsBytes); ok {
		return fmt.Errorf("proto: reflection requires map-backed extensions")
	}
	epb, err := extendable(pb)
	if err != nil {
		return err
	}
	registered := RegisteredExtensions(pb)
	for n := range fields {
		if registered[n] != nil || !isExtensionField(epb, n) {
			return fmt.Errorf("proto: invalid unregistered extension %d", n)
		}
	}
	existing := epb.extensionsWrite()
	for n := range existing {
		if registered[n] == nil {
			delete(existing, n)
		}
	}
	for n, b := range fields {
		existing[n] = Extension{enc: append([]byte(nil), b...)}
	}
	return nil
}

// MaterializeExtensionWire converts an already decoded map-backed extension to
// its current wire representation before an opt-in decoder merges another
// occurrence. It does not alter default legacy decoding behavior. Callers must
// synchronize writes and use the registered native descriptor.
func MaterializeExtensionWire(pb Message, desc *ExtensionDesc) error {
	if _, ok := pb.(extensionsBytes); ok {
		return fmt.Errorf("proto: reflection requires map-backed extensions")
	}
	epb, err := extendable(pb)
	if err != nil {
		return err
	}
	if err = checkExtensionTypes(epb, desc); err != nil {
		return err
	}
	fields, mu := epb.extensionsRead()
	if fields == nil {
		return nil
	}
	mu.Lock()
	defer mu.Unlock()
	e, ok := fields[desc.Field]
	if !ok || e.value == nil {
		return nil
	}
	e.enc = nil
	if err = e.Encode(); err != nil {
		return err
	}
	fields[desc.Field] = Extension{enc: append([]byte(nil), e.enc...)}
	return nil
}
