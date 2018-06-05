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
