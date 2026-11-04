package compare

import d "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"

func isFloatField(field *d.FieldDescriptorProto) bool {
	return field.GetType() == d.FieldDescriptorProto_TYPE_FLOAT || field.GetType() == d.FieldDescriptorProto_TYPE_DOUBLE
}

// Match sort.Float64Slice: NaNs precede numbers and tie with other NaNs.
// A tie must continue with the next element/field, rather than returning early.
func (p *plugin) generateFloatCompare(left, right string) {
	p.P("if ", left, " != ", right, " {")
	p.In()
	p.P("if aNaN, bNaN := ", p.mathPkg.Use(), ".IsNaN(float64(", left, ")), ", p.mathPkg.Use(), ".IsNaN(float64(", right, ")); aNaN != bNaN {")
	p.P("if aNaN { return -1 }; return 1")
	p.P("}")
	p.P("if ", left, " < ", right, " { return -1 }")
	p.P("if ", left, " > ", right, " { return 1 }")
	p.Out()
	p.P("}")
}
