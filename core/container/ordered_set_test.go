package container_test

import (
	"github.com/stretchr/testify/assert"
	"snow/core/container"
	"testing"
)

func TestOrderedSet(t *testing.T) {
	intSet := container.NewOrderedSet(4, 2, 1, 3)

	newSet := &container.OrderedSet[int]{}
	intSet.ScanIf(func(elem int) bool {
		if elem > 2 {
			newSet.Add(elem)
		}

		return true
	})
	assert.Equal(t, container.NewOrderedSet(4, 3), newSet, "ScanIf can scan all elements that satisfy predicts")

	newSet = &container.OrderedSet[int]{}
	intSet.Scan(func(elem int) {
		newSet.Add(elem)
	})

	assert.Equal(t, container.NewOrderedSet(4, 2, 1, 3), newSet, "Scan can scan all elements that satisfy predicts")

	assert.True(t, intSet.Len() == 4, "The length of ordered set must match the initializer arguments length specified")

	assert.True(t, intSet.Contains(1), "Set should contain elements as initializer specified")
	assert.False(t, intSet.Contains(-1), "Set should contain elements as initializer specified")

	newSet = container.NewOrderedSet(4, 2, 1, 3)
	newSet.Remove(1)
	assert.Equal(t, container.NewOrderedSet(4, 2, 3), newSet, "Remove ordered set element then left value must match specified")
	newSet.Remove(2)
	assert.Equal(t, container.NewOrderedSet(4, 3), newSet, "Remove ordered set element then left value must match specified")
	newSet.Remove(3)
	assert.Equal(t, container.NewOrderedSet(4), newSet, "Remove ordered set element then left value must match specified")
	newSet.Remove(4)
	assert.Empty(t, newSet, "Remove ordered set element then left value must match specified")

	newSet = container.NewOrderedSet(1, 2, 3, 4)
	newSet.Clear()
	assert.Empty(t, newSet, "Ordered set clear then set must match empty")

	newSet = container.NewOrderedSet(4, 2)
	otherSet := container.NewOrderedSet(1, 3)
	newSet.Union(otherSet)
	assert.Equal(t, intSet, newSet, "Give two ordered set and union them, result should match specified")

	newSet = container.NewOrderedSet(4, 2, 1, 3)
	otherSet = container.NewOrderedSet(1, 2, 3, 4, 5, 6)
	newSet.Intersect(otherSet)
	assert.Equal(t, intSet, newSet, "Give two ordered set and intersect them, result should match specified")

	newSet = container.NewOrderedSet(4, 2, 1, 3, 5, 6)
	otherSet = container.NewOrderedSet(5, 6)
	newSet.Substract(otherSet)
	assert.Equal(t, intSet, newSet, "Give two ordered set and intersect them, result should match specified")

	newSet = container.NewOrderedSet(4, 2)
	otherSet = container.NewOrderedSet(1, 3)
	result := newSet.Unioned(otherSet)
	assert.Equal(t, intSet, result, "Give two ordered set and unioned them, result should match specified")
	assert.Equal(t, container.NewOrderedSet(4, 2), newSet, "Give two ordered set and unioned them, origin should not change")

	newSet = container.NewOrderedSet(4, 2, 1, 3)
	otherSet = container.NewOrderedSet(1, 2, 3, 4, 5, 6)
	result = newSet.Intersected(otherSet)
	assert.Equal(t, intSet, result, "Give two ordered set and intersected them, result should match specified")
	assert.Equal(t, intSet, newSet, "Give two ordered set and intersected them, origin should not change")

	newSet = container.NewOrderedSet(4, 2, 1, 3, 5, 6)
	otherSet = container.NewOrderedSet(5, 6)
	result = newSet.Substracted(otherSet)
	assert.Equal(t, intSet, result, "Give two ordered set and substracted them, result should match specified")
	assert.Equal(t, container.NewOrderedSet(4, 2, 1, 3, 5, 6), newSet, "Give two ordered set and substracted them, origin should not change")

	foundKey := -1
	assert.True(t, intSet.WithKey(1, func(key int) { foundKey = key }) == 1, "Give one set and make use of WithKey() member")
	assert.True(t, foundKey == 1, "WithKey() should match specified")

	newSet = &container.OrderedSet[int]{}
	intSet.WithKeyRange(2, 4, func(key int) {
		if elem, ok := intSet.GetAt(key); ok {
			newSet.Add(elem)
		}
	})
	assert.Equal(t, container.NewOrderedSet(3, 4), newSet, "Give two ordered set and intersected them, result should match specified")
}
