package container_test

import (
	"github.com/mogud/snow/core/container"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSet(t *testing.T) {

	intSet := container.NewSet(4, 2, 1, 3)

	newList := container.NewList[int]()
	intSet.ScanIf(func(elem int) bool {
		if elem > 2 {
			newList.Add(elem)
		}
		return true
	})
	assert.ElementsMatch(t, container.NewList(4, 3), newList, "ScanIf can scan all elements that satisfy predicts")

	newList = container.NewList[int]()
	intSet.Scan(func(elem int) {
		newList.Add(elem)
	})

	assert.ElementsMatch(t, container.NewList(4, 2, 1, 3), newList, "Scan can scan all elements that satisfy predicts")

	assert.True(t, intSet.Len() == 4, "The length of ordered set must match the initializer arguments length specified")

	assert.True(t, intSet.Contains(1), "Set should contain elements as initializer specified")
	assert.False(t, intSet.Contains(-1), "Set should contain elements as initializer specified")

	newSet := container.NewSet(4, 2, 1, 3)
	newSet.Remove(1)
	assert.EqualValues(t, container.NewSet(4, 2, 3), newSet, "Remove set element then left value must match specified")
	newSet.Remove(2)
	assert.EqualValues(t, container.NewSet(4, 3), newSet, "Remove set element then left value must match specified")
	newSet.Remove(3)
	assert.EqualValues(t, container.NewSet(4), newSet, "Remove set element then left value must match specified")
	newSet.Remove(4)
	assert.Empty(t, newSet, "Remove set element then left value must match specified")

	newSet = container.NewSet(1, 2, 3, 4)
	newSet.Clear()
	assert.Empty(t, newSet, "Set clear then set must match empty")

	newSet = container.NewSet(4, 2)
	otherSet := container.NewSet(1, 3)
	newSet.Union(otherSet)
	assert.EqualValues(t, intSet, newSet, "give two set and intersect them, result should match specified")

	newSet = container.NewSet(4, 2, 1, 3)
	otherSet = container.NewSet(1, 2, 3, 4, 5, 6)
	newSet.Intersect(otherSet)
	assert.EqualValues(t, intSet, newSet, "Give two set and intersect them, result should match specified")

	newSet = container.NewSet(4, 2, 1, 3, 5, 6)
	otherSet = container.NewSet(5, 6)
	newSet.Substract(otherSet)
	assert.EqualValues(t, intSet, newSet, "Give two set and intersect them, result should match specified")

	newSet = container.NewSet(4, 2)
	otherSet = container.NewSet(1, 3)
	result := newSet.Unioned(otherSet)
	assert.EqualValues(t, intSet, result, "Give two set and unioned them, result should match specified")
	assert.EqualValues(t, container.NewSet(4, 2), newSet, "Give two  set and unioned them, origin should not change")

	newSet = container.NewSet(4, 2, 1, 3)
	otherSet = container.NewSet(1, 2, 3, 4, 5, 6)
	result = newSet.Intersected(otherSet)
	assert.EqualValues(t, intSet, result, "Give two  set and intersected them, result should match specified")
	assert.EqualValues(t, intSet, newSet, "Give two  set and intersected them, origin should not change")

	newSet = container.NewSet(4, 2, 1, 3, 5, 6)
	otherSet = container.NewSet(5, 6)
	result = newSet.Substracted(otherSet)
	assert.EqualValues(t, intSet, result, "Give two  set and substracted them, result should match specified")
	assert.EqualValues(t, container.NewSet(4, 2, 1, 3, 5, 6), newSet, "Give two  set and substracted them, origin should not change")

	foundKey := -1
	assert.True(t, intSet.WithKey(1, func(key int) { foundKey = key }) == 1, "Give one set and make use of WithKey() member")
	assert.True(t, foundKey == 1, "WithKey() should match specified")
}
