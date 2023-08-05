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
	Description string   `msg:"description,omitempty" json:"description,omitempty"`
	Tags        []string `msg:"tags,omitempty" json:"tags,omitempty"`
	exas.Exif
}

type Aggregate struct {
	Start    time.Time `msg:"start,omitempty" json:"start,omitempty"`
	End      time.Time `msg:"end,omitempty" json:"end,omitempty"`
	Location string    `msg:"location,omitempty" json:"location,omitempty"`
	Cover    string    `msg:"cover,omitempty" json:"cover,omitempty"`
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

func ReplaceTags(tags []string) MetadataAction {
	return func(instance Metadata) Metadata {
		instance.Tags = tags

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
