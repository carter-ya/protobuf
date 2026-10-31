package proto_test

import (
	"github.com/gogo/protobuf/proto"
	"testing"
	"time"
)

type legacyNativeClock struct {
	Time time.Time `protobuf:"-"`
}
type legacyClockMessage struct {
	Clock legacyNativeClock `protobuf:"bytes,1,opt,name=clock"`
}

var legacyClockMerge proto.InternalMessageInfo

func (m *legacyClockMessage) Reset()                      { *m = legacyClockMessage{} }
func (m *legacyClockMessage) String() string              { return "clock" }
func (*legacyClockMessage) ProtoMessage()                 {}
func (m *legacyClockMessage) XXX_Merge(src proto.Message) { legacyClockMerge.Merge(m, src) }
func TestCachedMergeNativeTime(t *testing.T) {
	value := time.Date(2026, 10, 31, 12, 0, 0, 123, time.FixedZone("test", 3600))
	source := &legacyClockMessage{Clock: legacyNativeClock{Time: value}}
	cloned := proto.Clone(source).(*legacyClockMessage)
	if !cloned.Clock.Time.Equal(value) || cloned.Clock.Time.Location().String() != value.Location().String() {
		t.Fatal("native time clone changed")
	}
	proto.Merge(cloned, &legacyClockMessage{})
	if !cloned.Clock.Time.Equal(value) {
		t.Fatal("unset nonnullable clock cleared destination")
	}
	source.Clock.Time = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	proto.Merge(cloned, source)
	if cloned.Clock.Time != source.Clock.Time {
		t.Fatal("native time merge retained private fields from the old value")
	}
}
