package filesystem

import (
	"strings"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func lessString(first, second string) bool {
	return strings.Compare(strings.ToLower(first), strings.ToLower(second)) < 0
}

func moreTime(first, second time.Time) bool {
	return first.After(second)
}

// ByHybridSort implements Sorter by type, name then modification time
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

	if first.IsDir && second.IsDir {
		return lessString(first.Name, second.Name)
	}

	if first.IsDir {
		return true
	}

	if second.IsDir {
		return false
	}

	if first.IsImage() && second.IsImage() {
		return moreTime(first.Date, second.Date)
	}

	if first.IsImage() {
		return false
	}

	if second.IsImage() {
		return true
	}

	return lessString(first.Name, second.Name)
}
