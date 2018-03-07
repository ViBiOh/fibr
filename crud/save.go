package crud

import (
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/ViBiOh/fibr/provider"
	"github.com/ViBiOh/fibr/utils"
)

const (
	defaultMaxMemory = 32 << 20 // 32 MB
)

func createOrOpenFile(filename string, info os.FileInfo) (io.WriteCloser, error) {
	if info == nil {
		return os.Create(filename)
	}
	return os.Open(filename)
}

// CreateDir creates given path directory to filesystem
func (a *App) CreateDir(w http.ResponseWriter, r *http.Request, config *provider.RequestConfig) {
	if !config.CanEdit {
		a.renderer.Error(w, http.StatusForbidden, errors.New(`You're not authorized to do this ⛔`))
		return
	}

	var filename string

	formName := r.FormValue(`name`)
	if formName != `` {
		filename, _ = utils.GetPathInfo(a.rootDirectory, config.Root, config.Path, formName)
	}

	if filename == `` {
		if !strings.HasSuffix(config.Path, `/`) {
			a.renderer.Error(w, http.StatusForbidden, errors.New(`You're not authorized to do this ⛔`))
			return
		}

		filename, _ = utils.GetPathInfo(a.rootDirectory, config.Root, config.Path)
	}

	if strings.Contains(filename, `..`) {
		a.renderer.Error(w, http.StatusForbidden, errors.New(`You're not authorized to do this ⛔`))
		return
	}

	if err := os.MkdirAll(filename, 0700); err != nil {
		a.renderer.Error(w, http.StatusInternalServerError, fmt.Errorf(`Error while creating directory: %v`, err))
		return
	}

	a.GetDir(w, config, path.Dir(filename), r.URL.Query().Get(`d`), &provider.Message{Level: `success`, Content: fmt.Sprintf(`Directory %s successfully created`, path.Base(filename))})
}

func (a *App) saveUploadedFile(config *provider.RequestConfig, uploadedFile io.ReadCloser, uploadedFileHeader *multipart.FileHeader) (string, error) {
	filename, info := utils.GetPathInfo(a.rootDirectory, config.Root, config.Path, uploadedFileHeader.Filename)
	hostFile, err := createOrOpenFile(filename, info)
	if hostFile != nil {
		defer func() {
			if err := hostFile.Close(); err != nil {
				log.Printf(`Error while closing writted file: %v`, err)
			}
		}()
	}

	if err != nil {
		return ``, fmt.Errorf(`Error while creating or opening file: %v`, err)
	} else if _, err = io.Copy(hostFile, uploadedFile); err != nil {
		return ``, fmt.Errorf(`Error while writing file: %v`, err)
	} else if provider.ImageExtensions[path.Ext(uploadedFileHeader.Filename)] {
		go a.generateImageThumbnail(strings.TrimPrefix(filename, a.rootDirectory))
	}

	return filename, nil
}

// SaveFiles saves form files to filesystem
func (a *App) SaveFiles(w http.ResponseWriter, r *http.Request, config *provider.RequestConfig) {
	if !config.CanEdit {
		a.renderer.Error(w, http.StatusForbidden, errors.New(`You're not authorized to do this ⛔`))
	}

	if err := r.ParseMultipartForm(defaultMaxMemory); err != nil {
		a.renderer.Error(w, http.StatusBadRequest, fmt.Errorf(`Error while parsing form: %v`, err))
		return
	}

	if r.MultipartForm.File == nil || len(r.MultipartForm.File[`files[]`]) == 0 {
		a.renderer.Error(w, http.StatusBadRequest, errors.New(`No file provided for save`))
		return
	}

	var outputDir string
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

		filename, err := a.saveUploadedFile(config, uploadedFile, file)
		if err != nil {
			a.renderer.Error(w, http.StatusInternalServerError, err)
			return
		}

		if outputDir == `` {
			outputDir = path.Dir(filename)
		}

		filenames[index] = file.Filename
	}

	message := fmt.Sprintf(`File %s successfully uploaded`, filenames[0])
	if len(filenames) > 1 {
		message = fmt.Sprintf(`Files %s successfully uploaded`, strings.Join(filenames, `, `))
	}

	a.GetDir(w, config, outputDir, r.URL.Query().Get(`d`), &provider.Message{Level: `success`, Content: message})
}
