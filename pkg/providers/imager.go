package providers

import "github.com/kube-project/receiver-service/models"

// ImageProvider defines functions which are used to handle images.
type ImageProvider interface {
	SaveImage(image *models.Image) (*models.Image, error)
	LoadImage(i int64) (*models.Image, error)
}
