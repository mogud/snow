package math

import (
	"github.com/mogud/snow/core/constraints"
)

func Clamp[T constraints.Ordered](v T, minV T, maxV T) T {
	switch {
	case v < minV:
		return minV
	case v > maxV:
		return maxV
	default:
		return v
	}
}

func Abs[T constraints.Integer | constraints.Float](a T) T {
	if a < 0 {
		return -a
	}
	return a
}
