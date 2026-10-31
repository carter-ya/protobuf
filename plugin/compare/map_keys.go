package compare

import (
	"strings"

	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
)

// Compare map keys and values in one deterministic order. Map lookup alone
// confuses absent keys with present zero values and iteration order is unstable.
// This emits the loop header; generateField emits the existing value comparison.
func (p *plugin) mapKeyOrder(field *descriptor.FieldDescriptorProto, fieldname string) {
	mapping := p.GoMapType(nil, field)
	keyType, _ := p.GoType(nil, mapping.KeyField)
	aliasType, _ := p.GoType(nil, mapping.KeyAliasField)
	keyType = strings.TrimPrefix(keyType, "*")
	aliasType = strings.TrimPrefix(aliasType, "*")
	left, right := "keysThis"+fieldname, "keysThat"+fieldname
	for _, side := range []struct{ receiver, keys string }{{"this", left}, {"that1", right}} {
		p.P(side.keys, ` := make([]`, keyType, `, 0, len(`, side.receiver, `.`, fieldname, `))`)
		p.P(`for key := range `, side.receiver, `.`, fieldname, ` { `, side.keys, ` = append(`, side.keys, `, `, keyType, `(key)) }`)
		p.P(p.sortkeysPkg.Use(), `.`, generator.CamelCase(keyType), `s(`, side.keys, `)`)
	}
	p.P(`for index, key := range `, left, ` {`)
	p.In()
	p.P(`if key != `, right, `[index] {`)
	if mapping.KeyField.IsBool() {
		p.P(`if !key { return -1 }`)
	} else {
		p.P(`if key < `, right, `[index] { return -1 }`)
	}
	p.P(`return 1`)
	p.P(`}`)
	p.P(`i := `, aliasType, `(key)`)
	p.Out()
}
