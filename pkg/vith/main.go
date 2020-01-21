package vith

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/ViBiOh/fibr/pkg/sha"
	"github.com/ViBiOh/httputils/v3/pkg/httperror"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
)

// App of package
type App interface {
	Handler() http.Handler
}

// Handler for request. Should be use with net/http
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := sha.Sha1(time.Now())
		inputName := fmt.Sprintf("/tmp/input_%s.jpeg", name)
		outputName := fmt.Sprintf("/tmp/output_%s.jpeg", name)
		copyBuffer := make([]byte, 32*1024)

		inputFile, err := os.OpenFile(inputName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			httperror.InternalServerError(w, err)
			return
		}

		if _, err := io.CopyBuffer(inputFile, r.Body, copyBuffer); err != nil {
			httperror.InternalServerError(w, err)
			return
		}

		cmd := exec.Command("ffmpeg", "-i", inputName, "-vf", "thumbnail", "-frames:v", "1", outputName)

		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out

		if err := cmd.Run(); err != nil {
			httperror.InternalServerError(w, err)
			logger.Error("%s", out.String())
			return
		}

		logger.Info("%s", out.String())

		thumbnail, err := os.OpenFile(outputName, os.O_RDONLY, 0600)
		if err != nil {
			httperror.InternalServerError(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)
		if _, err := io.CopyBuffer(w, thumbnail, copyBuffer); err != nil {
			logger.Error("%s", err)
		}

		if err := os.Remove(outputName); err != nil {
			logger.Error("%s", err)
		}
	})
}
