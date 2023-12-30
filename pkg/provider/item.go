package provider

import absto "github.com/ViBiOh/absto/pkg/model"

func KeepOnlyDir(items []absto.Item) []absto.Item {
	var filtered []absto.Item

	for _, item := range items {
		if item.IsDir() {
			filtered = append(filtered, item)
		}
	}

	return filtered
}

func KeepOnlyFile(items []absto.Item) []absto.Item {
	var filtered []absto.Item

	for _, item := range items {
		if !item.IsDir() {
			filtered = append(filtered, item)
		}
	}

	return filtered
}
