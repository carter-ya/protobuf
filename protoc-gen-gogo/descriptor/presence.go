package descriptor

// IsRealOneof reports whether a field belongs to a user-declared oneof.
// Proto3 optional's synthetic oneofs remain in descriptors, but must not
// produce a user-facing oneof API.
func (f *FieldDescriptorProto) IsRealOneof() bool {
	return f != nil && f.OneofIndex != nil && !f.GetProto3Optional()
}

// IsSyntheticOneof identifies an optional marker, not a one-field real oneof.
func (m *DescriptorProto) IsSyntheticOneof(index int32) bool {
	if m == nil || index < 0 || int(index) >= len(m.OneofDecl) {
		return false
	}
	for _, f := range m.Field {
		if f.GetProto3Optional() && f.OneofIndex != nil && *f.OneofIndex == index {
			return true
		}
	}
	return false
}

// HasProto3Optional reports whether this file needs an optional-aware runtime.
func (f *FileDescriptorProto) HasProto3Optional() bool {
	var visit func([]*DescriptorProto) bool
	visit = func(messages []*DescriptorProto) bool {
		for _, m := range messages {
			for _, field := range m.Field {
				if field.GetProto3Optional() {
					return true
				}
			}
			if visit(m.NestedType) {
				return true
			}
		}
		return false
	}
	return f != nil && visit(f.MessageType)
}
