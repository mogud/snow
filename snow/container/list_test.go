package container_test

import (
	"gitee.com/mogud/snow/container"
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestList(t *testing.T) {
	intList := container.NewList(1, 2, 3, 4)

	resultList := make([]int, 0, intList.Len())
	intList.ScanIf(func(elem int) bool {
		if elem > 2 {
			return false
		}
		resultList = append(resultList, elem)

		return true
	})

	assert.True(
		t,
		len(resultList) == 2,
		"ScanIf iterates all elements that satisfy specified predicate and expect result list len 2, but get:%d",
		len(resultList),
	)

	assert.Equal(
		t,
		resultList,
		[]int{1, 2},
		"ScanIf iterates all elements that satisfy specified predicate and expect result list:{1, 2}, but get:%v",
		resultList,
	)

	resultList = resultList[:0]
	intList.Scan(func(elem int) {
		resultList = append(resultList, elem)
	})

	assert.Condition(
		t,
		func() bool { return len(resultList) == 4 },
		"ScanIf iterates all elements that satisfy specified predicate and expect result list len 4, but get:%d",
		len(resultList),
	)

	assert.Equal(
		t,
		resultList,
		[]int{1, 2, 3, 4},
		"ScanIf iterates all elements that satisfy specified predicate and expect result list:{1, 2}, but get:%v",
		resultList,
	)

	assert.True(
		t,
		intList.Len() == 4,
		"The length of the list must match the initializer arguments length specified, but get:%d",
		intList.Len(),
	)

	copyList := intList.Copy()
	assert.Equal(
		t,
		intList,
		copyList,
		"Copy returns a list equals to the origin one, bug get:%+v",
		copyList,
	)

	copyList.Add(5)
	assert.Equal(
		t,
		copyList,
		container.NewList(1, 2, 3, 4, 5),
		"Add element to list expect{1,2,3,4,5}, but get:%v",
		copyList,
	)

	copyList = intList.Copy()
	copyList.AddIndex(1, 5)
	assert.Equal(
		t,
		copyList,
		container.NewList(1, 5, 2, 3, 4),
		"Add element to list at index 1, expect result list:{1,5, 2, 3, 4}, but get:%v",
		copyList,
	)

	assert.Panics(t, func() { copyList.AddIndex(-1, 1) }, "index is out of list")

	copyList = intList.Copy()
	copyList.Swap(0, 1)
	assert.Equal(
		t,
		copyList,
		container.NewList(2, 1, 3, 4),
		"swap list index = 0 and index = 1 element expect result{2,1,3,4}, but get:%v",
		copyList,
	)

	copyList = intList.Copy()
	copyList.RemoveIndex(0)
	assert.Equal(
		t,
		copyList,
		container.NewList(2, 3, 4),
		"swap list index = 0 and index = 1 element expect result{2,3,4}, but get:%v",
		copyList,
	)

	assert.Panics(t, func() { copyList.RemoveIndex(10) }, "index is out of list")

	copyList = intList.Copy()
	copyList.Clear()
	assert.Emptyf(t, copyList, "Clear all elements,list is not empty")

	copyList = intList.Copy()
	oldCopy := copyList
	copyList.Shuffle()
	assert.ElementsMatchf(t, copyList, intList, "Shuffle all elements get new list")
	assert.NotSame(t, copyList, oldCopy, "Shuffle get new list")

	copyList = intList.Copy()
	newList := copyList.Shuffled()
	assert.ElementsMatchf(t, copyList, newList, "Shuffle all elements get new list")
	assert.Equal(t, copyList, intList, "Shuffled the origin not change")

	assert.Truef(t, container.ListContains(intList, 1), "Check element in list")
	assert.Truef(t, !container.ListContains(intList, 20), "Check element not in list")

	assert.Truef(t, container.ListBinaryContains(intList, 1), "Binary contains check  element in list")
	assert.Truef(t, !container.ListBinaryContains(intList, 10), "Binary contains check  element  not in list")

	assert.Truef(t, !container.ListBinaryContains(intList, 10), "Binary contains check  element  not in list")

	assert.Truef(t, container.ListBinarySearch(intList, 1) == 0, "search element in list")
	assert.Truef(t, container.ListBinarySearch(intList, 10) == -1, "search element not in list")
}

func TestSort(t *testing.T) {
	intList := container.NewList(3, 1, 4, 2)
	resultList := container.NewList(1, 2, 3, 4)

	container.SortBy(intList, func(lhs, rhs int) bool {
		return lhs < rhs
	})
	assert.Equal(t, intList, resultList, "sort list by func(a,b int){ return a > b}")

	intList = container.NewList(3, 1, 4, 2)
	container.Sort(intList)
	assert.Equal(t, intList, resultList, "sort list")

	intList = container.NewList(3, 1, 4, 2)
	sortedList := container.SortedBy(intList, func(lhs, rhs int) bool {
		return lhs < rhs
	})
	assert.Equal(t, sortedList, resultList, "sorted list by func(a,b int){ return a > b}")
	assert.NotEqual(t, intList, sortedList, "sorted list not change origin list")

	intList = container.NewList(3, 1, 4, 2)
	sortedList = container.Sorted(intList)
	assert.Equal(t, sortedList, resultList, "sorted list")
	assert.NotEqual(t, intList, sortedList, "sorted list not change origin list")

	type object struct {
		index int
		value string
	}

	objList := container.NewList(
		object{1, "a"},
		object{2, "b"},
		object{4, "d"},
		object{1, "c"},
		object{1, "d"},
		object{1, "f"},
	)

	oldObjList := objList.Copy()

	newSortedList := container.SortedStableBy(objList, func(lhs, rhs object) bool {
		return lhs.index < rhs.index
	})

	assert.ElementsMatchf(t, oldObjList, newSortedList, "sorted list is element is match")

	for i := 0; i < 10; i++ {
		sortedList := container.SortedStableBy(objList, func(lhs, rhs object) bool {
			return lhs.index < rhs.index
		})
		assert.Equal(t, newSortedList, sortedList, "stable sorted result same")
	}
}
