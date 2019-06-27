package filesystem

import (
	"strings"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func lessString(first, second string) bool {
	return strings.Compare(strings.ToLower(first), strings.ToLower(second)) < 0
}

func lessTime(first, second time.Time) bool {
	return first.Before(second)
}

// ByName implements Sorter by name
type ByName []*provider.StorageItem

func (a ByName) Len() int {
	return len(a)
}

func (a ByName) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByName) Less(i, j int) bool {
	return lessString(a[i].Name, a[j].Name)
}

// ByModTime implements Sorter by modification time
type ByModTime []*provider.StorageItem

func (a ByModTime) Len() int {
	return len(a)
}

func (a ByModTime) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByModTime) Less(i, j int) bool {
	return lessTime(a[i].Date, a[j].Date)
}

// ByHybridSort implements Sorter by type then modification time
type ByHybridSort []*provider.StorageItem

func (a ByHybridSort) Len() int {
	return len(a)
}

func (a ByHybridSort) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByHybridSort) Less(i, j int) bool {
	first := a[i]
	second := a[j]

	if first.IsImage() == second.IsImage() {
		return lessTime(first.Date, second.Date)
	}

	return lessString(first.Name, second.Name)
}
