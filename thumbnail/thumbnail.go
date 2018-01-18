package thumbnail

import (
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"

	"github.com/nfnt/resize"
)

var tokenPool = make(chan int, 4)

func getToken() {
	tokenPool <- 1
}

func releaseToken() {
	<-tokenPool
}

func getThumbnail(filename string, width, height uint) (*image.Image, string, error) {
	reader, err := os.Open(filename)
	if reader != nil {
		defer reader.Close()
	}
	if err != nil {
		return nil, ``, fmt.Errorf(`Error while reading image: %v`, err)
	}

	img, imgType, err := image.Decode(reader)
	if err != nil {
		return nil, ``, fmt.Errorf(`Error while decoding image: %v`, err)
	}

	thumbnail := resize.Thumbnail(width, height, img, resize.Bilinear)

	return &thumbnail, imgType, nil
}

// ServeThumbnail render thumbnail image of given filename
func ServeThumbnail(w http.ResponseWriter, filename string, width, height uint) error {
	getToken()
	defer releaseToken()

	thumbnail, imgType, err := getThumbnail(filename, width, height)
	if err != nil {
		return fmt.Errorf(`Error while generating thumbnail: %s`, err)
	}

	w.Header().Set(`Content-Type`, fmt.Sprintf(`image/%s`, imgType))
	if imgType == `jpeg` {
		if err := jpeg.Encode(w, *thumbnail, nil); err != nil {
			return fmt.Errorf(`Error while encoding thumbnail: %v`, err)
		}
	} else if imgType == `png` {
		if err := png.Encode(w, *thumbnail); err != nil {
			return fmt.Errorf(`Error while encoding thumbnail: %v`, err)
		}
	} else if imgType == `gif` {
		if err := gif.Encode(w, *thumbnail, nil); err != nil {
			return fmt.Errorf(`Error while encoding thumbnail: %v`, err)
		}
	}

	return nil
}
