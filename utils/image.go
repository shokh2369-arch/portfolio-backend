package utils

import (
	"context"
	"log"
	"os"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

var cld *cloudinary.Cloudinary

func InitCloudinary() {
	cloudinaryURL := os.Getenv("CLOUDINARY_URL")
	if cloudinaryURL == "" {
		log.Fatal("❌ CLOUDINARY_URL environment variable is not set")
	}

	var err error
	cld, err = cloudinary.NewFromURL(cloudinaryURL)
	if err != nil {
		log.Fatalf("❌ Failed to initialize Cloudinary: %v", err)
	}
	log.Println("✅ Cloudinary initialized successfully")
}

func UploadImage(localPath string) (string, error) {
	if cld == nil {
		InitCloudinary()
	}

	ctx := context.Background()
	res, err := cld.Upload.Upload(ctx, localPath, uploader.UploadParams{
		Folder: "golang_portfolio",
	})
	if err != nil {
		return "", err
	}
	return res.SecureURL, nil
}

func BuildURL(publicID string) (string, error) {
	if cld == nil {
		InitCloudinary()
	}

	image, err := cld.Image(publicID)
	if err != nil {
		return "", err
	}
	return image.String()
}
