// Package fieldmeta contains resolved field behavior without depending on
// descriptors, the generator, or the protobuf runtime.
package fieldmeta

type Features struct {
	FieldPresence    int32
	EnumType         int32
	RepeatedEncoding int32
	UTF8Validation   int32
	MessageEncoding  int32
	JSONFormat       int32
	NamingStyle      int32
	SymbolVisibility int32
	ProtoLimits      int32
}

func (f Features) HasPresence() bool { return f.FieldPresence == 1 || f.FieldPresence == 3 }
func (f Features) Required() bool    { return f.FieldPresence == 3 }
func (f Features) Packed() bool      { return f.RepeatedEncoding == 1 }
func (f Features) VerifyUTF8() bool  { return f.UTF8Validation == 2 }
func (f Features) ClosedEnum() bool  { return f.EnumType == 2 }
func (f Features) Delimited() bool   { return f.MessageEncoding == 2 }
