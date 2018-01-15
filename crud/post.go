package crud

import (
	"net/http"

	"github.com/ViBiOh/fibr/provider"
)

const maxPostMemory = 32 * 1024 * 2014 // 32 MB

// Post handle post from form
func (a *App) Post(w http.ResponseWriter, r *http.Request, config *provider.RequestConfig) {
	isMultipart := false
	isForm := false

	if r.ParseMultipartForm(maxPostMemory) == nil {
		isMultipart = true
	}

	if r.ParseForm() == nil {
		isForm = true
	}

	if isForm && !isMultipart {
		a.CreateShare(w, r, config)
	} else {
		a.SaveFile(w, r, config)
	}
}
