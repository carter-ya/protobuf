package stdtypes

import (
	"go/parser"
	"math/rand"
	"testing"
)

func TestNilTimestampGoStringRegression(t *testing.T) {
	for _, sample := range []*StdTypes{{}, NewPopulatedStdTypes(rand.New(rand.NewSource(11)), false)} {
		if _, err := parser.ParseExpr(sample.GoString()); err != nil {
			t.Fatalf("nil timestamp produced invalid Go: %v", err)
		}
	}
}
