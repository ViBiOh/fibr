package crud

import (
	"log"
	"os"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/utils"
	"github.com/disintegration/imaging"
)

func (a *App) getThumbnailPath(rootRelativeFilename string) string {
	return path.Join(a.rootDirectory, provider.MetadataDirectoryName, rootRelativeFilename)
}

func (a *App) generateImageThumbnail(rootRelativeFilename string) {
	filename, info := utils.GetPathInfo(a.rootDirectory, rootRelativeFilename)
	if info == nil {
		log.Printf(`[thumbnail] Image not found for %s`, rootRelativeFilename)
		return
	}

	thumbnail, thumbInfo := utils.GetPathInfo(a.getThumbnailPath(rootRelativeFilename))
	if thumbInfo != nil {
		log.Printf(`[thumbnail] Thumbnail already exists for %s`, rootRelativeFilename)
		return
	}

	sourceImage, err := imaging.Open(filename)
	if err != nil {
		log.Printf(`[thumbnail] Error while opening file %s: %v`, rootRelativeFilename, err)
		return
	}

	resizedImage := imaging.Fill(sourceImage, 150, 150, imaging.Center, imaging.Box)

	thumbnailDir := path.Dir(thumbnail)
	thumbnailDirInfo, err := os.Stat(thumbnailDir)
	if err != nil && !os.IsNotExist(err) {
		log.Printf(`[thumbnail] Error while getting info for directory %s: %v`, thumbnailDir, err)
		return
	}

	if thumbnailDirInfo == nil {
		if err = os.MkdirAll(thumbnailDir, 0700); err != nil {
			log.Printf(`[thumbnail] Error while creating directory %s: %v`, thumbnailDir, err)
			return
		}
	}

	if err = imaging.Save(resizedImage, thumbnail); err != nil {
		log.Printf(`[thumbnail] Error while saving file for %s: %v`, rootRelativeFilename, err)
		return
	}

	log.Printf(`[thumbnail] Generation success for %s`, rootRelativeFilename)
}
