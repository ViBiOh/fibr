package crud

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/sha"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

func (a *app) getCover(files []provider.StorageItem) map[string]interface{} {
	for _, file := range files {
		if file.IsVideo() {
			continue
		}

		if a.thumbnailApp.HasThumbnail(file) {
			return map[string]interface{}{
				"Img":       file,
				"ImgHeight": thumbnail.Height,
				"ImgWidth":  thumbnail.Width,
			}
		}
	}

	return nil
}

// List render directory web view of given dirPath
func (a *app) List(w http.ResponseWriter, request provider.Request, message renderer.Message) (string, int, map[string]interface{}, error) {
	files, err := a.storageApp.List(request.GetFilepath(""))
	if err != nil {
		return "", 0, nil, err
	}

	uri := request.URL("")

	items := make([]provider.RenderItem, len(files))
	for index, file := range files {
		items[index] = provider.RenderItem{
			ID:          sha.Sha1(file.Name),
			URI:         uri,
			StorageItem: file,
		}
	}

	content := map[string]interface{}{
		"Paths": getPathParts(uri),
		"Files": items,
		"Cover": a.getCover(files),

		"Request": request,
		"Message": message,
	}

	if request.CanShare {
		content["Shares"] = a.metadataApp.Dump()
	}

	return "files", http.StatusOK, content, nil
}

// Download content of a directory into a streamed zip
func (a *app) Download(w http.ResponseWriter, request provider.Request) {
	zipWriter := zip.NewWriter(w)
	defer func() {
		if err := zipWriter.Close(); err != nil {
			logger.Error("unable to close zip: %s", err)
		}
	}()

	filename := path.Base(request.Path)
	if filename == "/" && len(request.Share.ID) != 0 {
		filename = path.Base(path.Join(request.Share.RootName, request.Path))
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", filename))

	if err := a.zipFiles(request, zipWriter, ""); err != nil {
		a.rendererApp.Error(w, err)
	}
}

func (a *app) zipFiles(request provider.Request, zipWriter *zip.Writer, pathname string) error {
	files, err := a.storageApp.List(request.GetFilepath(pathname))
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir {
			if err := a.zipFiles(request, zipWriter, path.Join(pathname, file.Name)); err != nil {
				return err
			}
		} else if err := a.addFileToZip(zipWriter, file, pathname); err != nil {
			return err
		}
	}

	return nil
}

func (a *app) addFileToZip(zipWriter *zip.Writer, file provider.StorageItem, pathname string) error {
	header, err := zip.FileInfoHeader(file.Info.(os.FileInfo))
	if err != nil {
		return err
	}

	header.Name = path.Join(pathname, file.Name)
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	reader, err := a.storageApp.ReaderFrom(file.Pathname)
	if reader != nil {
		defer func() {
			if err := reader.Close(); err != nil {
				logger.Error("unable to close zip file: %s", err)
			}
		}()
	}
	if err != nil {
		return err
	}

	buffer := provider.BufferPool.Get().(*bytes.Buffer)
	defer provider.BufferPool.Put(buffer)

	_, err = io.CopyBuffer(writer, reader, buffer.Bytes())
	return err
}
