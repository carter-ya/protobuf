package qualify

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"regexp"
	"strings"
	"testing"

	jsonstandard "github.com/gogo/protobuf/jsonpb/standard"
	legacy "github.com/gogo/protobuf/proto"
	textstandard "github.com/gogo/protobuf/textpb/standard"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

type node struct {
	Children []*node          `protobuf:"bytes,10,rep,name=children,proto3"`
	Nodes    map[string]*node `protobuf:"bytes,11,rep,name=nodes,proto3" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	I        int32            `protobuf:"varint,1,opt,name=i,proto3"`
	U        uint64           `protobuf:"varint,2,opt,name=u,proto3"`
	F        float64          `protobuf:"fixed64,3,opt,name=f,proto3"`
	Text     string           `protobuf:"bytes,4,opt,name=text,proto3"`
	Data     []byte           `protobuf:"bytes,5,opt,name=data,proto3"`
	Flag     bool             `protobuf:"varint,6,opt,name=flag,proto3"`
	Child    *node            `protobuf:"bytes,7,opt,name=child,proto3"`
	List     []int32          `protobuf:"varint,8,rep,packed,name=list,proto3"`
	Values   map[string]int32 `protobuf:"bytes,9,rep,name=values,proto3" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
}

func (m *node) Reset()                    { *m = node{} }
func (m *node) String() string            { return legacy.CompactTextString(m) }
func (*node) ProtoMessage()               {}
func (*node) Descriptor() ([]byte, []int) { return compressed, []int{0} }

var compressed []byte
var messageDescriptor protoreflect.MessageDescriptor

func init() {
	field := func(name string, number int32, kind descriptorpb.FieldDescriptorProto_Type) *descriptorpb.FieldDescriptorProto {
		return &descriptorpb.FieldDescriptorProto{Name: proto.String(name), Number: proto.Int32(number), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: kind.Enum()}
	}
	child := field("child", 7, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE)
	child.TypeName = proto.String(".qualification.Node")
	list := field("list", 8, descriptorpb.FieldDescriptorProto_TYPE_INT32)
	list.Label = descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum()
	values := field("values", 9, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE)
	values.Label = descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum()
	values.TypeName = proto.String(".qualification.Node.ValuesEntry")
	children := field("children", 10, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE)
	children.TypeName = proto.String(".qualification.Node")
	children.Label = descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum()
	nodes := field("nodes", 11, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE)
	nodes.TypeName = proto.String(".qualification.Node.NodesEntry")
	nodes.Label = descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum()
	nodeValue := field("value", 2, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE)
	nodeValue.TypeName = proto.String(".qualification.Node")

	source := &descriptorpb.FileDescriptorProto{
		Name: proto.String("qualification.proto"), Package: proto.String("qualification"), Syntax: proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("Node"),
				Field: []*descriptorpb.FieldDescriptorProto{
					field("i", 1, descriptorpb.FieldDescriptorProto_TYPE_INT32),
					field("u", 2, descriptorpb.FieldDescriptorProto_TYPE_UINT64),
					field("f", 3, descriptorpb.FieldDescriptorProto_TYPE_DOUBLE),
					field("text", 4, descriptorpb.FieldDescriptorProto_TYPE_STRING),
					field("data", 5, descriptorpb.FieldDescriptorProto_TYPE_BYTES),
					field("flag", 6, descriptorpb.FieldDescriptorProto_TYPE_BOOL),
					child, list, values, children, nodes,
				},
				NestedType: []*descriptorpb.DescriptorProto{
					{Name: proto.String("ValuesEntry"), Options: &descriptorpb.MessageOptions{MapEntry: proto.Bool(true)}, Field: []*descriptorpb.FieldDescriptorProto{field("key", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING), field("value", 2, descriptorpb.FieldDescriptorProto_TYPE_INT32)}},
					{Name: proto.String("NodesEntry"), Options: &descriptorpb.MessageOptions{MapEntry: proto.Bool(true)}, Field: []*descriptorpb.FieldDescriptorProto{field("key", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING), nodeValue}},
				},
			},
		},
	}

	file, err := protodesc.NewFile(source, nil)
	if err != nil {
		panic(err)
	}
	messageDescriptor = file.Messages().Get(0)
	wire, err := proto.Marshal(source)
	if err != nil {
		panic(err)
	}
	var out bytes.Buffer
	writer := gzip.NewWriter(&out)
	if _, err := writer.Write(wire); err != nil {
		panic(err)
	}
	if err := writer.Close(); err != nil {
		panic(err)
	}
	compressed = out.Bytes()
}
func compare(t *testing.T, got *node, want *dynamicpb.Message) {
	t.Helper()
	wire, err := legacy.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	back := dynamicpb.NewMessage(messageDescriptor)
	if err := proto.Unmarshal(wire, back); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(want, back) {
		t.Fatalf("semantic difference: SDK %s / official %s", back, want)
	}
}
func FuzzText(f *testing.F) {
	for _, seed := range []string{"", `i:1`, `i:+1`, `i: - 1`, `i:\v1`, `i:\f1`, `list:[]`, `list:[1,2]`, `values:[{key:'a' value:1}]`, `values:{key:'a' key:'b' value:1}`, `data:'\0'`, `data:'\xA'`, `text:'\uD800'`, `text:'\uFFFD'`, `flag:True`, `child:{i:1}`, `children:[]`, `children:[{}, {i:2}]`, `nodes:[{key:"a" value:{i:2}}]`, `nodes:{key:"a"}`, `f:-0`, `f:00`, `flag:00`, `flag:0x1`, `i:0b1`, `i:1_0`, `f:nan`, `f:1e9999`} {
		f.Add(strings.ReplaceAll(strings.ReplaceAll(seed, `\v`, "\v"), `\f`, "\f"))
	}
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 4096 || strings.Count(input, "{")+strings.Count(input, "[") > 64 {
			t.Skip()
		}
		got := new(node)
		sdkErr := (&textstandard.Unmarshaler{RecursionLimit: 100}).UnmarshalString(input, got)
		want := dynamicpb.NewMessage(messageDescriptor)
		officialErr := (prototext.UnmarshalOptions{RecursionLimit: 100}).Unmarshal([]byte(oracleWhitespace(input)), want)
		if officialErr == nil && oracleGrammarViolation(input) {
			officialErr = fmt.Errorf("numeric literal forbidden by published grammar")
		}
		if (sdkErr == nil) != (officialErr == nil) {
			t.Fatalf("text acceptance differs for %q: SDK %v / official %v", input, sdkErr, officialErr)
		}
		if sdkErr != nil {
			return
		}
		compare(t, got, want)
		output, err := (&textstandard.Marshaler{RecursionLimit: 100}).MarshalToString(got)
		if err != nil {
			t.Fatal(err)
		}
		parsed := dynamicpb.NewMessage(messageDescriptor)
		if err := prototext.Unmarshal([]byte(output), parsed); err != nil {
			t.Fatalf("official rejects SDK text %q: %v", output, err)
		}
		if !proto.Equal(parsed, want) {
			t.Fatal("SDK text output changed the message")
		}
	})
}
func FuzzJSON(f *testing.F) {
	for _, seed := range []string{`{}`, `null`, `{"i":1}`, `{"i":1,"i":2}`, `{"child":null}`, `{"list":[]}`, `{"list":[null]}`, `{"f":"NaN"}`, `{"f":"nan"}`, `{"text":"\uD800"}`, `{"u":"18446744073709551615"}`, `{"values":{"x":1}}`, `{"children":[{}, {"i":2}], "nodes":{"a":{}}}`, `{"nodes":{"a":null}}`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 4096 || strings.Count(input, "{")+strings.Count(input, "[") > 64 {
			t.Skip()
		}
		got := new(node)
		sdkErr := jsonstandard.UnmarshalString(input, got)
		want := dynamicpb.NewMessage(messageDescriptor)
		officialErr := (protojson.UnmarshalOptions{RecursionLimit: 100}).Unmarshal([]byte(input), want)
		if (sdkErr == nil) != (officialErr == nil) {
			t.Fatalf("JSON acceptance differs for %q: SDK %v / official %v", input, sdkErr, officialErr)
		}
		if sdkErr != nil {
			return
		}
		compare(t, got, want)
		output, err := (&jsonstandard.Marshaler{}).MarshalToString(got)
		if err != nil {
			t.Fatal(err)
		}
		parsed := dynamicpb.NewMessage(messageDescriptor)
		if err := protojson.Unmarshal([]byte(output), parsed); err != nil {
			t.Fatalf("official rejects SDK JSON %q: %v", output, err)
		}
		if !proto.Equal(parsed, want) {
			t.Fatal("SDK JSON output changed the message")
		}
	})
}

// The published grammar permits vertical tab/form feed between tokens. Official
// Go v1.36.12 does not recognize them there; normalize only that lexical context
// for the oracle, preserving every quoted string and comment byte.
func oracleWhitespace(input string) string {
	output := []byte(input)
	var quote byte
	escaped, comment := false, false
	for i, ch := range output {
		if comment {
			if ch == '\n' {
				comment = false
			}
			continue
		}
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '#' {
			comment = true
			continue
		}
		if ch == '\'' || ch == '"' {
			quote = ch
			continue
		}
		if ch == '\v' || ch == '\f' {
			output[i] = ' '
		}
	}
	return string(output)
}

// Go's text decoder accepts octal-looking float integers as decimal numbers.
// The published text grammar explicitly forbids octal floats. Mask strings and
// comments so this exception applies only to this fixture's float field tokens.
var octalFloat = regexp.MustCompile(`\bf[ \t\r\n\v\f]*:[ \t\r\n\v\f]*(?:-[ \t\r\n\v\f]*)?0[0-7]+(?:$|[ \t\r\n\v\f,;}>])`)

var floatBool = regexp.MustCompile(`\bflag[ \t\r\n\v\f]*:[ \t\r\n\v\f]*[01][fF](?:$|[ \t\r\n\v\f,;}>])`)

func oracleGrammarViolation(input string) bool {
	masked := []byte(input)
	var quote byte
	escaped, comment := false, false
	for i, ch := range masked {
		if comment {
			masked[i] = ' '
			if ch == '\n' {
				comment = false
			}
			continue
		}
		if quote != 0 {
			masked[i] = ' '
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '#' {
			comment = true
			masked[i] = ' '
			continue
		}
		if ch == '\'' || ch == '"' {
			quote = ch
			masked[i] = ' '
		}
	}
	return octalFloat.Match(masked) || floatBool.Match(masked)
}
func TestOracleGrammarExceptions(t *testing.T) {
	for _, input := range []string{`f:00`, `child:{f:-012}`, "f: - #comment\n 012", `flag:0F`, `flag:1f`} {
		if !oracleGrammarViolation(input) {
			t.Fatalf("did not identify %q", input)
		}
	}
	for _, input := range []string{`text:'f:00'`, `data:"f:00"`, "# f:00\nf:1", `f:0`, `f:0.1`, `f:1e00`, `flag:0`, `text:"flag:1f"`} {
		if oracleGrammarViolation(input) {
			t.Fatalf("misidentified %q", input)
		}
	}
}
