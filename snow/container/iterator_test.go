package container_test

import (
	"gitee.com/mogud/snow/container"
	"github.com/stretchr/testify/assert"
	"strconv"
	"strings"
	"testing"
)

func TestIterator(t *testing.T) {
	intList := container.NewList(1, 2, 3, 4)
	intMap := container.NewMap(
		container.Pair[int, string]{1, "ab"},
		container.Pair[int, string]{2, "ac"},
		container.Pair[int, string]{3, "ad"},
		container.Pair[int, string]{4, "f"},
	)
	intOrderedMap := container.NewOrderedMap(
		container.Pair[int, string]{1, "ab"},
		container.Pair[int, string]{2, "ac"},
		container.Pair[int, string]{3, "ad"},
		container.Pair[int, string]{4, "f"},
	)
	intSet := container.NewSet(1, 2, 3, 4)
	intOrderedSet := container.NewOrderedSet(1, 2, 3, 4)

	/** any */
	assert.True(
		t,
		container.Any[int](intList, func(elem int) bool {
			if elem > 2 {
				return true
			}
			return false
		}),
		"Container list element iteration have elements that meet the given conditions",
	)

	assert.True(
		t,
		container.Any[container.Pair[int, string]](intMap,
			func(entry container.Pair[int, string]) bool {
				if strings.Contains(entry.Second, "a") {
					return true
				}
				return false
			}),
		"Container map element iteration have elements that meet the given conditions",
	)
	assert.True(
		t,
		container.Any[container.Pair[int, string]](intOrderedMap,
			func(entry container.Pair[int, string]) bool {
				if strings.Contains(entry.Second, "a") {
					return true
				}
				return false
			}),
		"Container ordered map element iteration have elements that meet the given conditions",
	)
	assert.True(
		t,
		container.Any[int](intSet,
			func(elem int) bool {
				if elem > 2 {
					return true
				}
				return false
			}),
		"Container set element iteration have elements that meet the given conditions",
	)

	assert.True(
		t,
		container.Any[int](intOrderedSet,
			func(elem int) bool {
				if elem > 2 {
					return true
				}
				return false
			}),
		"Container ordered set element iteration have elements that meet the given conditions",
	)

	/** all */
	assert.False(
		t,
		container.All[int](intList, func(elem int) bool {
			if elem > 2 {
				return true
			}
			return false
		}),
		"Container list element iteration all elements that meet the given conditions",
	)

	assert.False(
		t,
		container.All[container.Pair[int, string]](intMap,
			func(entry container.Pair[int, string]) bool {
				if strings.Contains(entry.Second, "a") {
					return true
				}
				return false
			}),
		"Container map element iteration all elements that meet the given conditions",
	)
	assert.False(
		t,
		container.All[container.Pair[int, string]](intOrderedMap,
			func(entry container.Pair[int, string]) bool {
				if strings.Contains(entry.Second, "a") {
					return true
				}
				return false
			}),
		"Container ordered map element iteration all elements that meet the given conditions",
	)
	assert.False(
		t,
		container.All[int](intSet,
			func(elem int) bool {
				if elem > 2 {
					return true
				}
				return false
			}),
		"Container set element iteration all elements that meet the given conditions",
	)

	assert.False(
		t,
		container.All[int](intOrderedSet,
			func(elem int) bool {
				if elem > 2 {
					return true
				}
				return false
			}),
		"Container ordered set element iteration all elements that meet the given conditions",
	)

	/*Scan ordered containers using iterators with indexes*/
	listResult := make(map[int]int)
	listExpect := map[int]int{0: 1, 1: 2, 2: 3, 3: 4}
	container.ScanWithIndex[int](intList, func(index int, elem int) {
		listResult[index] = elem
	})
	assert.EqualValues(t, listResult, listExpect, "containers  list using iterators with indexes")

	setResult := make(map[int]int)
	setExpect := map[int]int{0: 1, 1: 2, 2: 3, 3: 4}
	container.ScanWithIndex[int](intOrderedSet, func(index int, elem int) {
		setResult[index] = elem

	})
	assert.EqualValues(t, setResult, setExpect, "containers  set using iterators with indexes")

	expectMap := map[int]string{0: "ab", 1: "ac", 2: "ad", 3: "f"}
	resultMap := make(map[int]string)
	container.ScanWithIndex[container.Pair[int, string]](intOrderedMap, func(index int, elem container.Pair[int, string]) {
		resultMap[index] = elem.Second
	})
	assert.EqualValues(t, expectMap, resultMap, "containers  set using iterators with indexes")

	// Scan ordered containers using iterators with indexes and elements that meet the given conditions
	listResult = make(map[int]int)
	listExpect = map[int]int{0: 1, 1: 2}
	container.ScanIfWithIndex[int](intList, func(index int, elem int) bool {
		if elem > 2 {
			return false
		}
		listResult[index] = elem
		return true
	})
	assert.EqualValues(t, listResult, listExpect, "containers list scan using iterators with indexes and elements that meet the given conditions")

	setResult = make(map[int]int)
	setExpect = map[int]int{0: 1, 1: 2}
	container.ScanIfWithIndex[int](intOrderedSet, func(index int, elem int) bool {
		if elem > 2 {
			return false
		}
		setResult[index] = elem
		return true

	})
	assert.EqualValues(t, setResult, setExpect, "containers order set scan using iterators with indexes and elements that meet the given conditions")

	expectMap = map[int]string{0: "ab", 1: "ac", 2: "ad"}
	resultMap = make(map[int]string)
	container.ScanIfWithIndex[container.Pair[int, string]](intOrderedMap, func(index int, elem container.Pair[int, string]) bool {
		if strings.Contains(elem.Second, "a") {
			resultMap[index] = elem.Second
			return true
		}
		return false
	})
	assert.EqualValues(t, resultMap, expectMap, "containers order map scan using iterators with indexes and elements that meet the given conditions")

	// Container iterator fold
	listSum := 0
	listSum = container.Fold[int](intList, listSum, func(acc int, elem int) int {
		return acc + elem
	})
	assert.True(t, listSum == 10, "list iterator fold sum")

	newList := container.NewList[int]()
	expectList := container.NewList(1, 2)
	newList = container.Fold[int](intList, newList, func(acc container.List[int], elem int) container.List[int] {
		if elem < 3 {
			acc = append(acc, elem)
		}
		return acc
	})
	assert.EqualValues(t, newList, expectList, "list iterator fold match specified")

	// Container ordered set trans elements to list and the results must match specified
	expectStrList := container.NewList("1", "2", "3", "4")
	resultStrList := container.Trans[int](intOrderedSet, func(elem int) string {
		return strconv.Itoa(elem)
	})
	assert.EqualValues(t, expectStrList, resultStrList, "Container ordered set trans elements to list and the results must match specified")

	//
	resultList := container.Filter[int](intSet, func(elem int) bool {
		if elem > 2 {
			return false
		}
		return true
	})

	sum := 0
	resultList.Scan(func(elem int) {
		sum += elem
	})
	assert.True(t, sum == 3, "Container filter elements to list and the results must match specified")

	// Flatten iterator and the results must match specified
	srcList := container.NewList(container.NewList(1, 2), container.NewList(3, 4))
	expectStrList = container.NewList("1", "2", "3", "4")
	resultStrList = container.FlatTrans[container.List[int]](srcList, func(elem container.List[int]) container.Iterator[string] {
		stringList := container.NewList[string]()
		elem.Scan(func(elem int) {
			stringList.Add(strconv.Itoa(elem))
		})
		return stringList
	})
	assert.EqualValues(t, expectStrList, resultStrList, "Flatten iterator and the results must match specified")

	// GroupBy iterator and the results must match specified
	expectStrMap := container.NewMap(
		container.Pair[string, container.List[string]]{"best", container.NewList("1", "2")},
		container.Pair[string, container.List[string]]{"good", container.NewList("3", "4")},
	)
	resultStrMap := container.GroupBy[int](
		intList,
		func(elem int) string {
			if elem > 2 {
				return "good"
			}
			return "best"
		},
		func(elem int) string {
			return strconv.Itoa(elem)
		})

	assert.EqualValues(t, expectStrMap, resultStrMap, "GroupBy iterator and the results must match specified")

	// Containers iterator to list and the results must match specified
	assert.EqualValues(t, container.ListOf[int](intOrderedSet), intList, "Containers set iterator to list and the results must match specified")
	assert.EqualValues(t, container.ListOf[container.Pair[int, string]](intOrderedMap), container.NewList(
		container.Pair[int, string]{1, "ab"},
		container.Pair[int, string]{2, "ac"},
		container.Pair[int, string]{3, "ad"},
		container.Pair[int, string]{4, "f"},
	), "Containers map iterator to list and the results must match specified")

	assert.EqualValues(t,
		container.Zip[int, int](intList, intOrderedSet),
		container.NewList(
			container.Pair[int, int]{1, 1},
			container.Pair[int, int]{2, 2},
			container.Pair[int, int]{3, 3},
			container.Pair[int, int]{4, 4},
		),
		"Containers zip two iterator and the results must match specified",
	)

	assert.EqualValues(t, container.SetOf[int](intList), intSet, "Containers iterator to set and the results must match specified")

	assert.EqualValues(t,
		container.Zip[int, int](intList, intOrderedSet),
		container.NewList(
			container.Pair[int, int]{1, 1},
			container.Pair[int, int]{2, 2},
			container.Pair[int, int]{3, 3},
			container.Pair[int, int]{4, 4},
		),
		"Containers zip two iterator and the results must match specified",
	)

	assert.EqualValues(t,
		container.MapOf[int, string](container.NewList(
			container.Pair[int, string]{1, "ab"},
			container.Pair[int, string]{2, "ac"},
			container.Pair[int, string]{3, "ad"},
			container.Pair[int, string]{4, "f"},
		)),
		intMap,
		"Containers iterator to map and the results must match specified",
	)

	assert.EqualValues(t,
		container.OrderedMapOf[int, string](container.NewList(
			container.Pair[int, string]{1, "ab"},
			container.Pair[int, string]{2, "ac"},
			container.Pair[int, string]{3, "ad"},
			container.Pair[int, string]{4, "f"},
		)),
		intOrderedMap,
		"Containers iterator to ordered map and the results must match specified",
	)

	assert.EqualValues(t,
		container.NewMap[int, string](
			container.Pair[int, string]{1, "1"},
			container.Pair[int, string]{2, "2"},
			container.Pair[int, string]{3, "3"},
			container.Pair[int, string]{4, "4"},
		),
		container.MapBy[int](intList, func(elem int) string {
			return strconv.Itoa(elem)
		}),
		"Map by iterator and the results must match specified",
	)

	assert.EqualValues(t,
		container.NewOrderedMap(
			container.Pair[int, string]{1, "1"},
			container.Pair[int, string]{2, "2"},
			container.Pair[int, string]{3, "3"},
			container.Pair[int, string]{4, "4"},
		),
		container.OrderedMapBy[int](intList, func(elem int) string {
			return strconv.Itoa(elem)
		}),
		"Ordered map by iterator and the results must match specified",
	)
}
