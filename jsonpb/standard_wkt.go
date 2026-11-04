package jsonpb

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
)

var durationJSON = regexp.MustCompile(`^(-?)([0-9]+)(\.([0-9]{1,9}))?s$`)
var timestampJSON = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?(Z|[+-][0-9]{2}:[0-9]{2})$`)

func parseStandardDuration(input string) (int64, int64, error) {
	parts := durationJSON.FindStringSubmatch(input)
	if parts == nil {
		return 0, 0, fmt.Errorf("invalid Protobuf Duration %q", input)
	}
	seconds, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || seconds > maxSecondsInDuration {
		return 0, 0, fmt.Errorf("Duration seconds out of range")
	}
	var nanos int64
	if parts[4] != "" {
		nanos, err = strconv.ParseInt(parts[4]+strings.Repeat("0", 9-len(parts[4])), 10, 32)
		if err != nil {
			return 0, 0, err
		}
	}
	if parts[1] == "-" {
		seconds = -seconds
		nanos = -nanos
	}
	return seconds, nanos, nil
}

func parseStandardTimestamp(input string) (*types.Timestamp, error) {
	if !timestampJSON.MatchString(input) {
		return nil, fmt.Errorf("invalid Protobuf Timestamp %q", input)
	}
	if input[len(input)-1] != 'Z' {
		zone := input[len(input)-6:]
		hours, _ := strconv.Atoi(zone[1:3])
		minutes, _ := strconv.Atoi(zone[4:6])
		if hours > 23 || minutes > 59 {
			return nil, fmt.Errorf("invalid Timestamp offset")
		}
	}
	value, err := time.Parse(time.RFC3339Nano, input)
	if err != nil {
		return nil, err
	}
	return types.TimestampProto(value)
}

func fieldMaskSnake(path string) string {
	var out strings.Builder
	for _, c := range path {
		if c >= 'A' && c <= 'Z' {
			out.WriteByte('_')
			out.WriteRune(c + 'a' - 'A')
		} else {
			out.WriteRune(c)
		}
	}
	return out.String()
}
func fieldMaskCamel(path string) string {
	var out strings.Builder
	upper := false
	for _, c := range path {
		if c == '_' {
			upper = true
			continue
		}
		if upper && c >= 'a' && c <= 'z' {
			c += 'A' - 'a'
		}
		out.WriteRune(c)
		upper = false
	}
	return out.String()
}
func marshalStandardFieldMask(mask *types.FieldMask) (string, error) {
	paths := make([]string, len(mask.Paths))
	for i, path := range mask.Paths {
		if path == "" || strings.Contains(path, ",") {
			return "", fmt.Errorf("invalid FieldMask path %q", path)
		}
		paths[i] = fieldMaskCamel(path)
		if fieldMaskSnake(paths[i]) != path {
			return "", fmt.Errorf("FieldMask path cannot round trip: %q", path)
		}
	}
	return strings.Join(paths, ","), nil
}
func unmarshalStandardFieldMask(input string) ([]string, error) {
	if input == "" {
		return nil, nil
	}
	paths := strings.Split(input, ",")
	for i, path := range paths {
		if path == "" || strings.Contains(path, "_") {
			return nil, fmt.Errorf("invalid JSON FieldMask path %q", path)
		}
		paths[i] = fieldMaskSnake(path)
	}
	return paths, nil
}

func standardCustomJSON(message proto.Message) bool {
	if _, ok := message.(*types.FieldMask); ok {
		return true
	}
	if value, ok := message.(isWkt); ok {
		return value.XXX_WellKnownType() != "Empty"
	}
	return false
}
func (m *jsonEncoder) marshalStandardSpecial(out *errWriter, message proto.Message) (bool, error) {
	if mask, ok := message.(*types.FieldMask); ok {
		text, err := marshalStandardFieldMask(mask)
		if err != nil {
			return true, err
		}
		encoded, err := json.Marshal(text)
		if err == nil {
			out.write(string(encoded))
			err = out.err
		}
		return true, err
	}
	if timestamp, ok := message.(*types.Timestamp); ok {
		_, err := types.TimestampFromProto(timestamp)
		if err != nil {
			return true, err
		}
	}
	return false, nil
}
func (u *jsonDecoder) unmarshalStandardSpecial(target reflect.Value, input json.RawMessage) (bool, error) {
	if mask, ok := target.Addr().Interface().(*types.FieldMask); ok {
		if len(input) == 0 || input[0] != '"' {
			return true, fmt.Errorf("expected FieldMask string")
		}
		var text string
		if err := json.Unmarshal(input, &text); err != nil {
			return true, err
		}
		paths, err := unmarshalStandardFieldMask(text)
		if err == nil {
			mask.Paths = paths
		}
		return true, err
	}
	return false, nil
}
