package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/NYTimes/gziphandler"
	"github.com/ViBiOh/auth/auth"
	authProvider "github.com/ViBiOh/auth/provider"
	"github.com/ViBiOh/auth/provider/basic"
	authService "github.com/ViBiOh/auth/service"
	"github.com/ViBiOh/fibr/crud"
	"github.com/ViBiOh/fibr/provider"
	"github.com/ViBiOh/fibr/ui"
	"github.com/ViBiOh/httputils"
	"github.com/ViBiOh/httputils/healthcheck"
	"github.com/ViBiOh/httputils/owasp"
)

func handleAnonymousRequest(w http.ResponseWriter, r *http.Request, err error, crudApp *crud.App, uiApp *ui.App) {
	if auth.IsForbiddenErr(err) {
		uiApp.Error(w, http.StatusForbidden, errors.New(`You're not authorized to do this ⛔️`))
	} else if !crudApp.CheckAndServeSEO(w, r) {
		if err == authProvider.ErrMalformedAuth || err == authProvider.ErrUnknownAuthType {
			uiApp.Error(w, http.StatusBadRequest, err)
		} else {
			w.Header().Add(`WWW-Authenticate`, `Basic`)
			uiApp.Error(w, http.StatusUnauthorized, err)
		}
	}
}

func browserHandler(crudApp *crud.App, uiApp *ui.App, authApp *auth.App) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodDelete {
			uiApp.Error(w, http.StatusMethodNotAllowed, errors.New(`We don't understand what you want from us`))
			return
		}

		if strings.Contains(r.URL.Path, `..`) {
			uiApp.Error(w, http.StatusForbidden, errors.New(`You're not authorized to do this ⛔`))
		}

		_, err := authApp.IsAuthenticated(r)

		config := &provider.RequestConfig{
			URL:      r.URL.Path,
			Path:     r.URL.Path,
			CanEdit:  true,
			CanShare: true,
		}

		if share := crudApp.GetSharedPath(config.Path); share != nil {
			config.Root = share.Path
			config.Path = strings.TrimPrefix(config.Path, fmt.Sprintf(`/%s`, share.ID))
			config.Prefix = share.ID

			if err != nil {
				config.CanEdit = share.Edit
				config.CanShare = false
			}

			err = nil
		}

		if err != nil {
			handleAnonymousRequest(w, r, err, crudApp, uiApp)
		} else if r.Method == http.MethodGet {
			crudApp.Get(w, r, config, nil)
		} else if r.Method == http.MethodPost {
			crudApp.Post(w, r, config)
		} else if r.Method == http.MethodPut {
			crudApp.CreateDir(w, r, config)
		} else if r.Method == http.MethodDelete {
			crudApp.Delete(w, r, config)
		} else {
			httputils.NotFound(w)
		}
	})
}

func main() {
	owaspConfig := owasp.Flags(``)
	authConfig := auth.Flags(`auth`)
	basicConfig := basic.Flags(`basic`)
	crudConfig := crud.Flags(``)
	uiConfig := ui.Flags(``)

	httputils.StartMainServer(func() http.Handler {
		authApp := auth.NewApp(authConfig, authService.NewBasicApp(basicConfig))
		uiApp := ui.NewApp(uiConfig, *crudConfig[`directory`])
		crudApp := crud.NewApp(crudConfig, uiApp)

		serviceHandler := owasp.Handler(owaspConfig, browserHandler(crudApp, uiApp, authApp))
		healthHandler := healthcheck.Handler()

		return gziphandler.GzipHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == `/health` {
				healthHandler.ServeHTTP(w, r)
			} else {
				serviceHandler.ServeHTTP(w, r)
			}
		}))
	}, nil)
}
