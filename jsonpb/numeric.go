package jsonpb

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var jsonIntegerNumber = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?$`)

// integerDecimal normalizes a JSON number without using floating point or
// allocating storage proportional to its exponent. The final range check is
// performed by strconv at the destination's width.
func integerDecimal(number string) (string, error) {
	bad := func() (string, error) { return "", fmt.Errorf("invalid integer %q", number) }
	if !jsonIntegerNumber.MatchString(number) {
		return bad()
	}
	negative := strings.HasPrefix(number, "-")
	mantissa := strings.TrimPrefix(number, "-")
	exponentText := "0"
	if i := strings.IndexAny(mantissa, "eE"); i >= 0 {
		exponentText = mantissa[i+1:]
		mantissa = mantissa[:i]
	}
	fraction := 0
	if i := strings.IndexByte(mantissa, '.'); i >= 0 {
		fraction = len(mantissa) - i - 1
		mantissa = mantissa[:i] + mantissa[i+1:]
	}
	digits := strings.TrimLeft(mantissa, "0")
	if digits == "" {
		if negative {
			return "-0", nil
		}
		return "0", nil
	}
	exponent, err := strconv.ParseInt(exponentText, 10, 32)
	if err != nil {
		return bad()
	}
	shift := exponent - int64(fraction)
	if shift < 0 {
		drop := -shift
		if drop > int64(len(digits)) {
			return bad()
		}
		for _, digit := range digits[len(digits)-int(drop):] {
			if digit != '0' {
				return bad()
			}
		}
		digits = digits[:len(digits)-int(drop)]
	} else {
		if int64(len(digits))+shift > 20 {
			return bad()
		}
		digits += strings.Repeat("0", int(shift))
	}
	if negative {
		digits = "-" + digits
	}
	return digits, nil
}
