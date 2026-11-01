// Compare the additive HYBRID surface with the pinned official Go generator.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
)

func surface(path string) map[string]string {
	fs := token.NewFileSet()
	f, e := parser.ParseFile(fs, path, nil, 0)
	if e != nil {
		panic(e)
	}
	out := map[string]string{}
	render := func(n ast.Node) string {
		var b bytes.Buffer
		if e := format.Node(&b, fs, n); e != nil {
			panic(e)
		}
		return strings.Join(strings.Fields(b.String()), " ")
	}
	for _, d := range f.Decls {
		switch x := d.(type) {
		case *ast.FuncDecl:
			if x.Recv == nil {
				continue
			}
			r := render(x.Recv.List[0].Type)
			name := x.Name.Name
			if !(r == "*Node" || r == "*Cold" || r == "Node_builder" || r == "Cold_builder" || strings.HasPrefix(r, "case_Node_")) {
				continue
			}
			if !(strings.HasPrefix(name, "Set") || strings.HasPrefix(name, "Has") || strings.HasPrefix(name, "Clear") || strings.HasPrefix(name, "Which") || name == "ProtoReflect" || name == "Build" || name == "String" && strings.HasPrefix(r, "case_")) {
				continue
			}
			for _, list := range []*ast.FieldList{x.Type.Params, x.Type.Results} {
				if list != nil {
					for _, field := range list.List {
						field.Names = nil
					}
				}
			}
			out[r+"."+name] = render(x.Type)
		case *ast.GenDecl:
			for _, s := range x.Specs {
				switch s := s.(type) {
				case *ast.TypeSpec:
					n := s.Name.Name
					if n == "Node_builder" || n == "Cold_builder" || strings.HasPrefix(n, "case_Node_") {
						out[n] = render(s.Type)
					}
				case *ast.ValueSpec:
					for i, n := range s.Names {
						if strings.HasPrefix(n.Name, "Node_") && strings.HasSuffix(n.Name, "_case") {
							out[n.Name] = render(s.Type) + "=" + render(s.Values[i])
						}
					}
				}
			}
		}
	}
	return out
}
func main() {
	if len(os.Args) != 3 {
		panic("expected official.go generated.go")
	}
	a, b := surface(os.Args[1]), surface(os.Args[2])
	if !reflect.DeepEqual(a, b) {
		aa, _ := json.MarshalIndent(a, "", "  ")
		bb, _ := json.MarshalIndent(b, "", "  ")
		fmt.Printf("official:\n%s\ngogo:\n%s\n", aa, bb)
		os.Exit(1)
	}
	fmt.Println("Official HYBRID API signatures match:", len(a))
}
