package crud

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/ViBiOh/fibr/utils"
	"github.com/ViBiOh/httputils"
)

const maxUploadSize = 32 * 1024 * 2014 // 32 MB

func getFileForm(w http.ResponseWriter, r *http.Request) (io.ReadCloser, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	uploadedFile, _, err := r.FormFile(`file`)
	if err != nil {
		return uploadedFile, fmt.Errorf(`Error while reading file form: %v`, err)
	}

	return uploadedFile, nil
}

func createOrOpenFile(filename string, info os.FileInfo) (io.WriteCloser, error) {
	if info == nil {
		return os.Create(filename)
	}
	return os.Open(filename)
}

func saveDir(w http.ResponseWriter, r *http.Request, filename string) {
	if err := os.MkdirAll(filename, 0700); err != nil {
		httputils.InternalServerError(w, fmt.Errorf(`Error while creating directory: %v`, err))
	} else {
		w.WriteHeader(http.StatusCreated)
	}
}

func saveFile(w http.ResponseWriter, r *http.Request, filename string, info os.FileInfo) {
	uploadedFile, err := getFileForm(w, r)
	if uploadedFile != nil {
		defer uploadedFile.Close()
	}
	if err != nil {
		httputils.BadRequest(w, fmt.Errorf(`Error while getting file from form: %v`, err))
		return
	}

	hostFile, err := createOrOpenFile(filename, info)
	if hostFile != nil {
		defer hostFile.Close()
	}
	if err != nil {
		httputils.InternalServerError(w, fmt.Errorf(`Error while creating or opening file: %v`, err))
		return
	}

	if _, err = io.Copy(hostFile, uploadedFile); err != nil {
		httputils.InternalServerError(w, fmt.Errorf(`Error while writing file: %v`, err))
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// Save given path to filesystem (create directory or file given on Path)
func Save(w http.ResponseWriter, r *http.Request, directory string) {
	filename, info := utils.GetPathInfo(directory, r.URL.Path)

	if strings.HasSuffix(r.URL.Path, `/`) {
		saveDir(w, r, filename)
		return
	} else if info != nil && info.IsDir() {
		httputils.Forbidden(w)
		return
	}

	saveFile(w, r, filename, info)
}
