package ui

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

// GeneratePageContent generate content for rendering page
func GeneratePageContent(baseContent map[string]interface{}, r *http.Request, current os.FileInfo, files []os.FileInfo) map[string]interface{} {
	pathParts := strings.Split(strings.Trim(r.URL.Path, `/`), `/`)
	if pathParts[0] == `` {
		pathParts = nil
	}

	seo := baseContent[`Seo`].(map[string]interface{})
	pageContent := map[string]interface{}{
		`Config`: baseContent[`Config`],
	}

	pageContent[`PathParts`] = pathParts
	pageContent[`Current`] = current
	pageContent[`Files`] = files
	pageContent[`Seo`] = map[string]interface{}{
		`Title`:       fmt.Sprintf(`fibr - %s`, r.URL.Path),
		`Description`: fmt.Sprintf(`FIle BRowser of directory %s on the server`, r.URL.Path),
		`URL`:         r.URL.Path,
		`Img`:         seo[`Img`],
		`ImgHeight`:   seo[`ImgHeight`],
		`ImgWidth`:    seo[`ImgWidth`],
	}

	params := r.URL.Query()
	if params.Get(`message_content`) != `` {
		pageContent[`Message`] = map[string]interface{}{
			`Level`:   params.Get(`message_level`),
			`Content`: params.Get(`message_content`),
		}
	}

	return pageContent
}
