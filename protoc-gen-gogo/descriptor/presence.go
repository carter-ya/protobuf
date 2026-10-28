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
