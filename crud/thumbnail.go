package crud

import (
	"log"
	"os"
	"path"

	"github.com/ViBiOh/fibr/provider"
	"github.com/ViBiOh/fibr/utils"
	"github.com/disintegration/imaging"
)

func (a *App) generateImageThumbnail(rootRelativeFilename string) {
	filename, info := utils.GetPathInfo(a.rootDirectory, rootRelativeFilename)
	thumbnail, thumbInfo := utils.GetPathInfo(a.rootDirectory, provider.MetadataDirectoryName, rootRelativeFilename)

	if info == nil {
		log.Printf(`[thumbnail] No origin file for %s`, filename)
	}

	if thumbInfo != nil {
		log.Printf(`[thumbnail] Already exist for %s`, filename)
		return
	}

	origImage, err := imaging.Open(filename)
	if err != nil {
		log.Printf(`[thumbnail] Error while opening origin file %s: %v`, filename, err)
		return
	}

	resizedImage := imaging.Resize(origImage, 150, 0, imaging.Box)

	if err = os.MkdirAll(path.Dir(thumbnail), 0700); err != nil {
		log.Printf(`[thumbnail] Error while creating directory %s: %v`, path.Dir(thumbnail), err)
		return
	}

	if err = imaging.Save(resizedImage, thumbnail); err != nil {
		log.Printf(`[thumbnail] Error while saving file for %s: %v`, filename, err)
		return
	}
}
