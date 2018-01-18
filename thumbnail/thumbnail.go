package thumbnail

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"
	"strconv"

	"github.com/nfnt/resize"
)

func getThumbnail(filename string, width, height uint) (*image.Image, string, error) {
	reader, err := os.Open(filename)
	if err != nil {
		return nil, ``, fmt.Errorf(`Error while reading image: %v`, err)
	}
	defer reader.Close()

	img, imgType, err := image.Decode(reader)
	if err != nil {
		return nil, ``, fmt.Errorf(`Error while decoding image: %v`, err)
	}

	thumbnail := resize.Thumbnail(width, height, img, resize.MitchellNetravali)

	return &thumbnail, imgType, nil
}

// ServeThumbnail service thumbnail image of given filename
func ServeThumbnail(w http.ResponseWriter, filename string, width, height uint) error {
	thumbnail, imgType, err := getThumbnail(filename, width, height)
	if err != nil {
		return fmt.Errorf(`Error while generating thumbnail: %s`, err)
	}

	buffer := new(bytes.Buffer)
	if imgType == `jpeg` {
		if err := jpeg.Encode(buffer, *thumbnail, nil); err != nil {
			return fmt.Errorf(`Error while encoding thumbnail: %v`, err)
		}
	} else if imgType == `png` {
		if err := png.Encode(buffer, *thumbnail); err != nil {
			return fmt.Errorf(`Error while encoding thumbnail: %v`, err)
		}
	} else if imgType == `gif` {
		if err := gif.Encode(buffer, *thumbnail, nil); err != nil {
			return fmt.Errorf(`Error while encoding thumbnail: %v`, err)
		}
	}

	w.Header().Set(`Content-Type`, fmt.Sprintf(`image/%s`, imgType))
	w.Header().Set(`Content-Length`, strconv.Itoa(len(buffer.Bytes())))
	if _, err := w.Write(buffer.Bytes()); err != nil {
		return fmt.Errorf(`Error while writing thumbnail: %s`, err)
	}

	return nil
}
