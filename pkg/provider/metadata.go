package provider

import (
	"context"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	exas "github.com/ViBiOh/exas/pkg/model"
)

//go:generate mockgen -source metadata.go -destination ../mocks/metadata.go -package mocks -mock_names MetadataManager=MetadataManager

type ExifResponse struct {
	Exif exas.Exif  `json:"exif"`
	Item absto.Item `json:"item"`
}

type Metadata struct {
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	exas.Exif
}

type Aggregate struct {
	Start    time.Time `json:"start,omitempty"`
	End      time.Time `json:"end,omitempty"`
	Location string    `json:"location,omitempty"`
	Cover    string    `json:"cover,omitempty"`
}

type MetadataAction func(Metadata) Metadata

func ReplaceExif(exif exas.Exif) MetadataAction {
	return func(instance Metadata) Metadata {
		instance.Exif = exif

		return instance
	}
}

func ReplaceDescription(description string) MetadataAction {
	return func(instance Metadata) Metadata {
		instance.Description = description

		return instance
	}
}

func AddTag(tag string) MetadataAction {
	return func(instance Metadata) Metadata {
		instance.Tags = append(instance.Tags, tag)

		return instance
	}
}

func RemoveTag(tag string) MetadataAction {
	return func(instance Metadata) Metadata {
		if index := findIndex(instance.Tags, tag); index != -1 {
			instance.Tags = append(instance.Tags[:index], instance.Tags[index+1:]...)
		}

		return instance
	}
}

type MetadataManager interface {
	ListDir(ctx context.Context, item absto.Item) ([]absto.Item, error)

	GetAggregateFor(ctx context.Context, item absto.Item) (Aggregate, error)
	GetAllAggregateFor(ctx context.Context, items ...absto.Item) (map[string]Aggregate, error)
	SaveAggregateFor(ctx context.Context, item absto.Item, aggregate Aggregate) error

	GetMetadataFor(ctx context.Context, item absto.Item) (Metadata, error)
	GetAllMetadataFor(ctx context.Context, items ...absto.Item) (map[string]Metadata, error)
	Update(ctx context.Context, item absto.Item, opts ...MetadataAction) (Metadata, error)
}
