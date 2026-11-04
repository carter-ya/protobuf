package types_test

import (
	"math"
	"sort"
	"testing"

	types "github.com/gogo/protobuf/types"
)

func TestFloatWrapperCompareOrdering(t *testing.T) {
	values := []float64{math.Float64frombits(0x7ff8000000000001), math.Float64frombits(0xfff8000000000002), math.Inf(-1), -1, math.Copysign(0, -1), 0, 1, math.Inf(1)}
	for _, a := range values {
		for _, b := range values {
			want := 0
			order := sort.Float64Slice{a, b}
			if order.Less(0, 1) {
				want = -1
			} else if order.Less(1, 0) {
				want = 1
			}
			left, right := &types.DoubleValue{Value: a}, &types.DoubleValue{Value: b}
			if got := left.Compare(right); got != want {
				t.Fatalf("double %v / %v: %d want %d", a, b, got, want)
			}
			fleft, fright := &types.FloatValue{Value: float32(a)}, &types.FloatValue{Value: float32(b)}
			if got := fleft.Compare(fright); got != want {
				t.Fatalf("float %v / %v: %d want %d", a, b, got, want)
			}
		}
	}
	left := &types.DoubleValue{Value: math.NaN(), XXX_unrecognized: []byte{16, 1}}
	right := &types.DoubleValue{Value: math.NaN(), XXX_unrecognized: []byte{16, 2}}
	if left.Compare(right) >= 0 || right.Compare(left) <= 0 {
		t.Fatal("NaN tie skipped subsequent unknown bytes")
	}
}
