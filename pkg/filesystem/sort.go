package filesystem

import (
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
)

// ByName implements Sorter by name
type ByName []*provider.StorageItem

func (a ByName) Len() int {
	return len(a)
}

func (a ByName) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByName) Less(i, j int) bool {
	return strings.Compare(strings.ToLower(a[i].Name), strings.ToLower(a[j].Name)) < 0
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
	return a[i].Date.Before(a[j].Date)
}
