package thumbnail

import (
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/disintegration/imaging"
)

var (
	ignoredThumbnailDir = map[string]bool{
		`vendor`:       true,
		`vendors`:      true,
		`node_modules`: true,
	}
)

// App stores informations
type App struct {
	storages []provider.Storage
}

// NewApp creates new App from Flags' config
func NewApp(storages []provider.Storage) *App {
	return &App{
		storages: storages,
	}
}

// Generate thumbnail for all storage
func (a App) Generate() {
	for _, storage := range a.storages {
		err := storage.Walk(func(pathname string, item *provider.StorageItem, _ error) error {
			if item.IsDir && strings.HasPrefix(item.Name, `.`) || ignoredThumbnailDir[item.Name] {
				return filepath.SkipDir
			}

			if provider.ImageExtensions[path.Ext(item.Name)] {
				info, err := storage.Info(path.Join(provider.MetadataDirectoryName, pathname))
				if err != nil && !os.IsNotExist(err) {
					return err
				}

				if info == nil {
					a.GenerateImageThumbnail(storage, pathname)
				}
			}

			return nil
		})

		if err != nil {
			log.Printf(`[thumbnail] Error while walking %s: %v`, storage.Name(), err)
		}
	}
}

// GenerateImageThumbnail generate thumbnail image for given path
func (a App) GenerateImageThumbnail(storage provider.Storage, pathname string) {
	info, err := storage.Info(pathname)
	if err != nil && !os.IsNotExist(err) {
		log.Printf(`[thumbnail] Error while getting info about %s: %v`, pathname, err)
		return
	}

	if info == nil {
		log.Printf(`[thumbnail] Image not found for %s`, pathname)
		return
	}

	thumbnailPath := path.Join(provider.MetadataDirectoryName, pathname)

	thumbInfo, err := storage.Info(thumbnailPath)
	if err != nil && !os.IsNotExist(err) {
		log.Printf(`[thumbnail] Error while getting info about thumbnail for %s: %v`, pathname, err)
		return
	}

	if thumbInfo != nil {
		log.Printf(`[thumbnail] Thumbnail already exists for %s`, pathname)
		return
	}

	file, err := storage.Read(pathname)
	if file != nil {
		defer func() {
			if err := file.Close(); err != nil {
				log.Printf(`[thumbnail] Error while closing file %s: %v`, pathname, err)
			}
		}()
	}
	if err != nil {
		log.Printf(`[thumbnail] Error while opening file %s: %v`, pathname, err)
		return
	}

	sourceImage, err := imaging.Decode(file)
	if err != nil {
		log.Printf(`[thumbnail] Error while opening file %s: %v`, pathname, err)
		return
	}
	resizedImage := imaging.Fill(sourceImage, 150, 150, imaging.Center, imaging.Box)

	if err := storage.Create(path.Dir(thumbnailPath)); err != nil {
		log.Printf(`[thumbnail] Error while getting creating thumbnail dir for %s: %v`, pathname, err)
		return
	}

	thumbnailFile, err := storage.Open(thumbnailPath)
	if thumbnailFile != nil {
		defer func() {
			if err := thumbnailFile.Close(); err != nil {
				log.Printf(`[thumbnail] Error while closing file %s: %v`, thumbnailPath, err)
			}
		}()
	}
	if err != nil {
		log.Printf(`[thumbnail] Error while opening thumbnail file %s: %v`, pathname, err)
		return
	}

	format, err := imaging.FormatFromFilename(thumbnailPath)
	if err != nil {
		log.Printf(`[thumbnail] Error while getting thumbnail format for %s: %v`, pathname, err)
		return
	}

	if err = imaging.Encode(thumbnailFile, resizedImage, format); err != nil {
		log.Printf(`[thumbnail] Error while saving file for %s: %v`, pathname, err)
		return
	}

	log.Printf(`[thumbnail] Generation success for %s`, pathname)
}
