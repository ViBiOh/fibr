package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	absto "github.com/ViBiOh/absto/pkg/model"
)

func LoadJSON[T any](ctx context.Context, storageApp absto.Storage, filename string) (output T, err error) {
	var reader io.ReadCloser

	reader, err = storageApp.ReadFrom(ctx, filename)
	if err != nil {
		err = fmt.Errorf("read: %w", err)
		return
	}

	defer func() {
		err = errors.Join(err, reader.Close())
	}()

	if err = json.NewDecoder(reader).Decode(&output); err != nil {
		err = fmt.Errorf("decode: %w", storageApp.ConvertError(err))
	}

	return
}

func SaveJSON[T any](ctx context.Context, storageApp absto.Storage, filename string, content T) error {
	reader, writer := io.Pipe()

	done := make(chan error)
	go func() {
		defer close(done)
		var err error

		if jsonErr := json.NewEncoder(writer).Encode(content); jsonErr != nil {
			err = fmt.Errorf("encode: %w", jsonErr)
		}

		done <- errors.Join(err, writer.Close())
	}()

	return errors.Join(storageApp.WriteTo(ctx, filename, reader, absto.WriteOpts{}), <-done)
}
