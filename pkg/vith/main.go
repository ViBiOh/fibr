package vith

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
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
		query := r.URL.Query()

		width := strings.TrimSpace(query.Get("width"))
		height := strings.TrimSpace(query.Get("height"))

		outputName := fmt.Sprintf("/tmp/%s.jpeg", sha.Sha1(time.Now()))

		cmd := exec.Command("ffmpeg", "-i", "pipe:0", "-vf", fmt.Sprintf("thumbnail,scale=%s:%s", width, height), "-frames:v", "1", outputName)
		cmd.Stdin = r.Body

		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out

		if err := cmd.Run(); err != nil {
			httperror.InternalServerError(w, err)
			logger.Error("%s", out.String())
			return
		}

		logger.Info("Thumbnail generated for %s", r.URL.Path)

		thumbnail, err := os.OpenFile(outputName, os.O_RDONLY, 0600)
		if err != nil {
			httperror.InternalServerError(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)
		if _, err := io.Copy(w, thumbnail); err != nil {
			logger.Error("%s", err)
		}

		if err := os.Remove(outputName); err != nil {
			logger.Error("%s", err)
		}
	})
}
