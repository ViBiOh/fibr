package thumbnail

import (
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"path"
	"strings"

	"github.com/disintegration/imaging"
)

var tokenPool = make(chan int, 2)

func getToken() {
	tokenPool <- 1
}

func releaseToken() {
	<-tokenPool
}

func getThumbnail(filename string, width, height int) (image.Image, error) {
	src, err := imaging.Open(filename)
	if err != nil {
		return nil, fmt.Errorf(`Error while reading image: %v`, err)
	}

	return imaging.Thumbnail(src, width, height, imaging.Box), nil
}

// ServeThumbnail render thumbnail image of given filename
func ServeThumbnail(w http.ResponseWriter, filename string, width, height int) error {
	getToken()
	defer releaseToken()

	thumbnail, err := getThumbnail(filename, width, height)
	if err != nil {
		return fmt.Errorf(`Error while generating thumbnail: %s`, err)
	}

	imgType := strings.TrimPrefix(path.Ext(filename), `.`)

	w.Header().Set(`Content-Type`, fmt.Sprintf(`image/%s`, imgType))
	if imgType == `jpeg` {
		if err := jpeg.Encode(w, thumbnail, nil); err != nil {
			return fmt.Errorf(`Error while encoding thumbnail: %v`, err)
		}
	} else if imgType == `png` {
		if err := png.Encode(w, thumbnail); err != nil {
			return fmt.Errorf(`Error while encoding thumbnail: %v`, err)
		}
	} else if imgType == `gif` {
		if err := gif.Encode(w, thumbnail, nil); err != nil {
			return fmt.Errorf(`Error while encoding thumbnail: %v`, err)
		}
	}

	return nil
}
