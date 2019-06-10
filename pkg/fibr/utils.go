package fibr

import (
	"net/http"
)

func isMethodAllowed(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet:
		fallthrough
	case http.MethodPost:
		fallthrough
	case http.MethodPut:
		fallthrough
	case http.MethodPatch:
		fallthrough
	case http.MethodDelete:
		return true
	default:
		return false
	}
}
