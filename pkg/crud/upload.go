package crud

import (
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"path"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
)

const (
	defaultMaxMemory = 32 << 20 // 32 MB
)

func (a *App) saveUploadedFile(request *provider.Request, uploadedFile io.ReadCloser, uploadedFileHeader *multipart.FileHeader) error {
	filename := provider.GetPathname(request, []byte(uploadedFileHeader.Filename))

	hostFile, err := a.storage.Open(filename)
	if hostFile != nil {
		defer func() {
			if err := hostFile.Close(); err != nil {
				log.Printf(`Error while closing writted file: %v`, err)
			}
		}()
	}

	if err != nil {
		return fmt.Errorf(`Error while creating or opening file: %v`, err)
	}

	if _, err = io.Copy(hostFile, uploadedFile); err != nil {
		return fmt.Errorf(`Error while writing file: %v`, err)
	}

	if provider.ImageExtensions[path.Ext(uploadedFileHeader.Filename)] {
		go a.thumbnailApp.GenerateImageThumbnail(filename)
	}

	return nil
}

// Upload saves form files to filesystem
func (a *App) Upload(w http.ResponseWriter, r *http.Request, request *provider.Request) {
	if !request.CanEdit {
		a.renderer.Error(w, http.StatusForbidden, ErrNotAuthorized)
		return
	}

	if err := r.ParseMultipartForm(defaultMaxMemory); err != nil {
		a.renderer.Error(w, http.StatusBadRequest, fmt.Errorf(`Error while parsing form: %v`, err))
		return
	}

	if r.MultipartForm.File == nil || len(r.MultipartForm.File[`files[]`]) == 0 {
		a.renderer.Error(w, http.StatusBadRequest, errors.New(`No file provided for save`))
		return
	}

	filenames := make([]string, len(r.MultipartForm.File[`files[]`]))

	for index, file := range r.MultipartForm.File[`files[]`] {
		uploadedFile, err := file.Open()
		if uploadedFile != nil {
			defer func() {
				if err := uploadedFile.Close(); err != nil {
					log.Printf(`Error while closing uploaded file: %v`, err)
				}
			}()
		}
		if err != nil {
			a.renderer.Error(w, http.StatusBadRequest, fmt.Errorf(`Error while getting file from form: %v`, err))
			return
		}

		if err := a.saveUploadedFile(request, uploadedFile, file); err != nil {
			a.renderer.Error(w, http.StatusInternalServerError, err)
			return
		}

		filenames[index] = file.Filename
	}

	message := fmt.Sprintf(`File %s successfully uploaded`, filenames[0])
	if len(filenames) > 1 {
		message = fmt.Sprintf(`Files %s successfully uploaded`, strings.Join(filenames, `, `))
	}

	a.List(w, request, request.Path, r.URL.Query().Get(`d`), &provider.Message{Level: `success`, Content: message})
}
