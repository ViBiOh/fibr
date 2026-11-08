package provider

import (
	"context"
	jsonv1 "encoding/json"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"

	absto "github.com/ViBiOh/absto/pkg/model"
)

func LoadJSON[T any](ctx context.Context, storageService absto.Storage, filename string) (output T, err error) {
	var reader io.ReadCloser

	reader, err = storageService.ReadFrom(ctx, filename)
	if err != nil {
		err = fmt.Errorf("read: %w", err)
		return output, err
	}

	defer func() {
		err = errors.Join(err, reader.Close())
	}()

	if err = json.UnmarshalRead(reader, &output, jsonv1.FormatDurationAsNano(true), json.FormatNilMapAsNull(true), json.FormatNilSliceAsNull(true)); err != nil {
		err = fmt.Errorf("decode: %w", storageService.ConvertError(err))
	}

	return output, err
}

func SaveJSON[T any](ctx context.Context, storageService absto.Storage, filename string, content T) error {
	reader, writer := io.Pipe()

	done := make(chan error)
	go func() {
		defer close(done)
		var err error

		if jsonErr := json.MarshalWrite(writer, content, jsonv1.FormatDurationAsNano(true), json.FormatNilMapAsNull(true), json.FormatNilSliceAsNull(true)); jsonErr != nil {
			err = fmt.Errorf("encode: %w", jsonErr)
		}

		done <- errors.Join(err, writer.Close())
	}()

	if err := storageService.WriteTo(ctx, filename, reader, absto.WriteOpts{}); err != nil {
		_ = reader.Close()

		for range done {
		}

		return err
	}

	return <-done
}
