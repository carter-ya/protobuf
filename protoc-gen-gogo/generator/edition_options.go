package generator

import (
	"fmt"
	"github.com/gogo/protobuf/proto"
	"strings"
)

// Enable options individually. Representation changes and fast codecs remain
// disabled until their own resolved-semantics contract is implemented.
func validateEditionOption(options proto.Message, extension *proto.ExtensionDesc) error {
	number := extension.Field
	if !strings.HasPrefix(extension.Name, "gogoproto.") && (number < 62000 || number >= 68000) {
		return nil
	}
	switch number {
	case 63017, 64017, 63018, 64018, 63020, 64020, 63028, 64028, 63001, 64001, 63002, 62001, 63003, 64003,
		63004, 64004, 63006, 64006, 63007, 64007, 63008, 67008,
		63013, 64013, 63029, 64029, 63021, 62021, 63022, 62022,
		62023, 66001, 65004, 65005, 65006, 63027, 63033, 64033,
		63034, 64034, 63035, 64035:
		return nil
	case 63026, 64026:
		if proto.GetBoolExtension(options, extension, true) {
			return nil
		}
		return fmt.Errorf("gogo option %d cannot discard Editions unknown fields", number)
	}
	return fmt.Errorf("gogo extension %d is not yet supported with Editions", number)
}
