package crud

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/ViBiOh/fibr/ui"
	"github.com/ViBiOh/fibr/utils"
)

const maxUploadSize = 32 * 1024 * 2014 // 32 MB

func getFileForm(w http.ResponseWriter, r *http.Request) (io.ReadCloser, *multipart.FileHeader, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	uploadedFile, uploadedFileHeader, err := r.FormFile(`file`)
	if err != nil {
		return uploadedFile, uploadedFileHeader, fmt.Errorf(`Error while reading file form: %v`, err)
	}

	return uploadedFile, uploadedFileHeader, nil
}

func createOrOpenFile(filename string, info os.FileInfo) (io.WriteCloser, error) {
	if info == nil {
		return os.Create(filename)
	}
	return os.Open(filename)
}

// CreateDir creates given path directory to filesystem
func CreateDir(w http.ResponseWriter, r *http.Request, directory string, uiConfig *ui.App) {
	if strings.HasSuffix(r.URL.Path, `/`) {
		filename, _ := utils.GetPathInfo(directory, r.URL.Path)

		if err := os.MkdirAll(filename, 0700); err != nil {
			uiConfig.Error(w, http.StatusInternalServerError, fmt.Errorf(`Error while creating directory: %v`, err))
		} else {
			w.WriteHeader(http.StatusCreated)
		}
	} else {
		uiConfig.Error(w, http.StatusForbidden, errors.New(`You're not authorized to do this`))
	}
}

// SaveFile saves form file to filesystem
func SaveFile(w http.ResponseWriter, r *http.Request, directory string, uiConfig *ui.App) {
	uploadedFile, uploadedFileHeader, err := getFileForm(w, r)
	if uploadedFile != nil {
		defer uploadedFile.Close()
	}
	if err != nil {
		uiConfig.Error(w, http.StatusBadRequest, fmt.Errorf(`Error while getting file from form: %v`, err))
		return
	}

	filename, info := utils.GetPathInfo(directory, r.URL.Path, uploadedFileHeader.Filename)
	hostFile, err := createOrOpenFile(filename, info)
	if hostFile != nil {
		defer hostFile.Close()
	}
	if err != nil {
		uiConfig.Error(w, http.StatusInternalServerError, fmt.Errorf(`Error while creating or opening file: %v`, err))
	} else if _, err = io.Copy(hostFile, uploadedFile); err != nil {
		uiConfig.Error(w, http.StatusInternalServerError, fmt.Errorf(`Error while writing file: %v`, err))
	} else {
		GetDir(w, path.Dir(filename), uiConfig, &ui.Message{Level: `success`, Content: fmt.Sprintf(`File %s successfully uploaded`, uploadedFileHeader.Filename)})
	}
}
