package crud

import (
	"archive/zip"
	"io"
	"net/http"
	"os"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
)

func (a *app) getCover(files []provider.StorageItem) map[string]interface{} {
	for _, file := range files {
		if a.thumbnail.HasThumbnail(file) {
			return map[string]interface{}{
				"Img":       file,
				"ImgHeight": thumbnail.ThumbnailHeight,
				"ImgWidth":  thumbnail.ThumbnailWidth,
			}
		}
	}

	return nil
}

// List render directory web view of given dirPath
func (a *app) List(w http.ResponseWriter, request provider.Request, message *provider.Message) {
	files, err := a.storage.List(request.GetFilepath(""))
	if err != nil {
		a.renderer.Error(w, request, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	content := map[string]interface{}{
		"Paths": getPathParts(request.GetURI("")),
		"Files": files,
		"Cover": a.getCover(files),
	}

	if request.CanShare {
		content["Shares"] = a.metadatas
	}

	a.renderer.Directory(w, request, content, message)
}

// Download content of a directory into a streamed zip
func (a *app) Download(w http.ResponseWriter, request provider.Request) {
	files, err := a.storage.List(request.GetFilepath(""))
	if err != nil {
		a.renderer.Error(w, request, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	zipWriter := zip.NewWriter(w)
	defer func() {
		if err := zipWriter.Close(); err != nil {
			logger.Error("unable to close zip: %s", err)
		}
	}()

	for _, file := range files {
		if file.IsDir {
			continue
		}

		if err = a.addFileToZip(zipWriter, file); err != nil {
			a.renderer.Error(w, request, provider.NewError(http.StatusInternalServerError, err))
			return
		}
	}
}

func (a *app) addFileToZip(zipWriter *zip.Writer, file provider.StorageItem) error {
	header, err := zip.FileInfoHeader(file.Info.(os.FileInfo))
	if err != nil {
		return err
	}

	header.Name = file.Name
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	reader, err := a.storage.ReaderFrom(file.Pathname)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, reader)
	return err
}
