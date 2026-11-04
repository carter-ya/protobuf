// Command compare-generated compares Go syntax and decoded descriptors. Go's
// formatter and gzip compressor change across toolchains; neither is a public
// generated API. Every declaration, field tag, method body and raw descriptor
// byte remains part of this comparison. Frozen fixtures are never rewritten.
package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"
)

func normalized(path string) ([]byte, error) {
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return nil, err
	}
	var decodeErr error
	ast.Inspect(f, func(node ast.Node) bool {
		v, ok := node.(*ast.ValueSpec)
		if !ok || len(v.Names) != 1 || !strings.HasPrefix(v.Names[0].Name, "fileDescriptor_") || len(v.Values) != 1 {
			return true
		}
		lit, ok := v.Values[0].(*ast.CompositeLit)
		if !ok {
			return true
		}
		var compressed []byte
		for _, elt := range lit.Elts {
			n, ok := elt.(*ast.BasicLit)
			if !ok {
				decodeErr = fmt.Errorf("unexpected descriptor literal")
				return false
			}
			value, err := strconv.ParseUint(n.Value, 0, 8)
			if err != nil {
				decodeErr = err
				return false
			}
			compressed = append(compressed, byte(value))
		}
		r, err := gzip.NewReader(bytes.NewReader(compressed))
		if err != nil {
			decodeErr = err
			return false
		}
		raw, err := ioutil.ReadAll(r)
		r.Close()
		if err != nil {
			decodeErr = err
			return false
		}
		lit.Elts = nil
		for _, b := range raw {
			lit.Elts = append(lit.Elts, &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(int(b))})
		}
		return false
	})
	if decodeErr != nil {
		return nil, decodeErr
	}
	var out bytes.Buffer
	posType := reflect.TypeOf(token.Pos(0))
	err = ast.Fprint(&out, token.NewFileSet(), f, func(name string, v reflect.Value) bool {
		return v.Type() != posType && name != "Obj" && name != "Scope" && name != "Unresolved"
	})
	return out.Bytes(), err
}

func main() {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: compare-generated frozen.go candidate.go")
		os.Exit(2)
	}
	a, err := normalized(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	b, err := normalized(args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !bytes.Equal(a, b) {
		fmt.Fprintf(os.Stderr, "generated API, code or decoded descriptor differs: %s %s\n", args[0], args[1])
		os.Exit(1)
	}
}
