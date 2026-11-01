package generator

import "strings"

func (g *Generator) generateReflectionMethods(mc *msgCtx, fields []topLevelField) {
	if !g.ReflectMessage(mc.message) {
		return
	}
	used := make(map[string]bool)
	for _, n := range methodNames {
		used[n] = true
	}
	for _, field := range fields {
		switch f := field.(type) {
		case *simpleField:
			used[f.goName] = true
			used[f.getterName] = true
		case *oneofField:
			used[f.goName] = true
			used[f.getterName] = true
			for _, s := range f.subFields {
				used[s.getterName] = true
			}
		}
	}
	claim := func(name string) {
		if used[name] {
			g.Fail("HYBRID/reflection method conflicts with existing Go API: ", mc.goName, ".", name)
		}
		used[name] = true
	}
	claim("ProtoReflect")
	claim("XXX_GogoReflect")
	runtime := g.reflectionRuntimeImport()
	api := g.AddImport(GoImportPath("google.golang.org/protobuf/reflect/protoreflect"))
	g.P("func (m *", mc.goName, ") ProtoReflect() ", api, ".Message { return ", runtime, ".MessageOf(m) }")
	g.P("func (*", mc.goName, ") XXX_GogoReflect() {}")
	if !g.Hybrid(mc.message) {
		return
	}
	for _, field := range fields {
		switch f := field.(type) {
		case *simpleField:
			claim("Set" + f.goName)
			t := f.goType
			pointer := strings.HasPrefix(t, "*") && f.protoField.IsScalar()
			if pointer {
				t = t[1:]
			}
			g.P("func (m *", mc.goName, ") Set", f.goName, "(v ", t, ") {")
			if pointer {
				g.P("m.", f.goName, " = &v")
			} else {
				if f.protoField.IsBytes() {
					g.P("if v == nil { v = make(", f.goType, ",0) }")
				}
				g.P("m.", f.goName, " = v")
			}
			g.P("}")
			if g.FieldHasPresence(mc.message.File(), f.protoField) {
				claim("Has" + f.goName)
				claim("Clear" + f.goName)
				g.P("func (m *", mc.goName, ") Has", f.goName, "() bool { return m != nil && m.", f.goName, " != nil }")
				g.P("func (m *", mc.goName, ") Clear", f.goName, "() { m.", f.goName, " = nil }")
			}
		case *oneofField:
			claim("Has" + f.goName)
			claim("Clear" + f.goName)
			claim("Which" + f.goName)
			g.P("func (m *", mc.goName, ") Has", f.goName, "() bool { return m != nil && m.", f.goName, " != nil }")
			g.P("func (m *", mc.goName, ") Clear", f.goName, "() { m.", f.goName, " = nil }")
			caseType := "case_" + mc.goName + "_" + f.goName
			unset := mc.goName + "_" + f.goName + "_not_set_case"
			g.P("type ", caseType, " ", api, ".FieldNumber")
			g.P("const ", unset, " ", caseType, " = 0")
			for _, s := range f.subFields {
				g.P("const ", mc.goName, "_", s.goName, "_case ", caseType, " = ", s.fieldNumber)
			}
			g.P("func (x ", caseType, ") String() string { if x == 0 { return \"not set\" }; fd := (*", mc.goName, ")(nil).ProtoReflect().Descriptor().Fields().ByNumber(", api, ".FieldNumber(x)); if fd != nil { return string(fd.Name()) }; return fmt.Sprintf(\"%d\",x) }")
			g.P("func (m *", mc.goName, ") Which", f.goName, "() ", caseType, " { if m == nil { return ", unset, " }; switch m.", f.goName, ".(type) {")
			for _, s := range f.subFields {
				g.P("case *", s.oneofTypeName, ": return ", mc.goName, "_", s.goName, "_case")
			}
			g.P("default: return ", unset, " } }")
			for _, s := range f.subFields {
				claim("Set" + s.goName)
				claim("Has" + s.goName)
				claim("Clear" + s.goName)
				g.P("func (m *", mc.goName, ") Set", s.goName, "(v ", s.goType, ") {")
				if s.protoField.IsMessage() || g.IsGroup(s.protoField) {
					g.P("if v == nil { m.", f.goName, " = nil; return }")
				}
				if s.protoField.IsBytes() {
					g.P("if v == nil { v = make(", s.goType, ",0) }")
				}
				g.P("m.", f.goName, " = &", s.oneofTypeName, "{", s.goName, ":v} }")
				g.P("func (m *", mc.goName, ") Has", s.goName, "() bool { if m == nil { return false }; _,ok := m.", f.goName, ".(*", s.oneofTypeName, "); return ok }")
				g.P("func (m *", mc.goName, ") Clear", s.goName, "() { if _,ok := m.", f.goName, ".(*", s.oneofTypeName, "); ok { m.", f.goName, " = nil } }")
			}
		}
	}
	g.P("type ", mc.goName, "_builder struct { _ [0]func()")
	for _, field := range fields {
		switch f := field.(type) {
		case *simpleField:
			g.P(f.goName, " ", f.goType)
		case *oneofField:
			for _, s := range f.subFields {
				typ := s.goType
				if s.protoField.IsScalar() && !s.protoField.IsBytes() {
					typ = "*" + typ
				}
				g.P(s.goName, " ", typ)
			}
		}
	}
	g.P("}")
	g.P("func (b ", mc.goName, "_builder) Build() *", mc.goName, " { m := &", mc.goName, "{}")
	for _, field := range fields {
		switch f := field.(type) {
		case *simpleField:
			g.P("m.", f.goName, " = b.", f.goName)
		case *oneofField:
			for _, s := range f.subFields {
				value := "b." + s.goName
				if s.protoField.IsScalar() && !s.protoField.IsBytes() {
					value = "*" + value
				}
				g.P("if b.", s.goName, " != nil { m.", f.goName, " = &", s.oneofTypeName, "{", s.goName, ":", value, "} }")
			}
		}
	}
	g.P("return m }")
}

// Legacy plugins allocate standard reflect through a separate import helper.
// Keep the optional runtime under an explicit, non-standard alias.
func (g *Generator) reflectionRuntimeImport() GoPackageName {
	path := GoImportPath("github.com/gogo/protobuf/reflect")
	if alias, ok := g.packageNames[path]; ok {
		g.addedImports[path] = true
		return alias
	}
	alias := GoPackageName("gogoreflect")
	for g.usedPackageNames[alias] || globalPackageNames[alias] {
		alias += "_"
	}
	g.packageNames[path] = alias
	g.usedPackageNames[alias] = true
	g.addedImports[path] = true
	return alias
}
