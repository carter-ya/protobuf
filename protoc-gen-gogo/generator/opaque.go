package generator

import "strings"

const opaqueFieldPrefix = "xxx_hidden_"

func opaqueFieldName(name string) string { return opaqueFieldPrefix + name }
func (g *Generator) publicFieldName(message *Descriptor, name string) string {
	if g.Opaque(message) {
		return strings.TrimPrefix(name, opaqueFieldPrefix)
	}
	return name
}

// Field indices follow the declaration order, including oneof interface slots.
// Internal legacy fields follow these slots and keep their historical names.
func (g *Generator) generatePrivateFieldAccessor(mc *msgCtx, fields []topLevelField) {
	if !g.Opaque(mc.message) {
		return
	}
	g.P("func (m *", mc.goName, ") XXX_GogoReflectField(index int) interface{} { switch index {")
	for index, field := range fields {
		switch f := field.(type) {
		case *simpleField:
			g.P("case ", index, ": return &m.", f.goName)
		case *oneofField:
			g.P("case ", index, ": return &m.", f.goName)
		}
	}
	g.P("default: return nil } }")
}

// HybridOutputFiles identifies outputs that require mutually exclusive builds.
func (g *Generator) HybridOutputFiles() []string {
	var names []string
	for _, file := range g.genFiles {
		for _, m := range file.Messages() {
			if !m.GetOptions().GetMapEntry() && g.Hybrid(m) {
				names = append(names, file.goFileName(g.pathType))
				break
			}
		}
	}
	return names
}
