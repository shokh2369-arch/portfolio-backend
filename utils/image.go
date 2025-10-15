package utils

import (
	"context"
	"log"
	"os"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

// Hardcoded Cloudinary URL
const cloudinaryURL = "cloudinary://129343476295679:4Qf5grKG_o2uY26vhT03KTwlHCc@diitl5gey"

// Global Cloudinary instance
var cld *cloudinary.Cloudinary

func init() {
	var err error
	cld, err = cloudinary.NewFromURL(cloudinaryURL)
	if err != nil {
		log.Fatalf("Failed to initialize Cloudinary: %v", err)
	}
}

// UploadImage uploads a local file to Cloudinary and returns the secure URL
func UploadImage(localPath string) (string, error) {
	ctx := context.Background()
	file, err := os.Open(localPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	res, err := cld.Upload.Upload(ctx, file, uploader.UploadParams{
		Folder: "golang_portfolio",
	})
	if err != nil {
		return "", err
	}
	return res.SecureURL, nil
}

// BuildURL returns the Cloudinary URL for a given public ID
func BuildURL(publicID string) (string, error) {
	image, err := cld.Image(publicID)
	if err != nil {
		return "", err
	}
	url, err := image.String()
	if err != nil {
		return "", err
	}
	return url, nil
}
