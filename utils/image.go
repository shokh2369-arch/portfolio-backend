package utils

import (
	"context"
	"log"
	"os"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/joho/godotenv"
)

func ImageUploadPath(photo string) string {
	_ = godotenv.Load() // loads .env if exists

	cld, err := cloudinary.NewFromURL(os.Getenv("CLOUDINARY_URL"))
	ctx := context.Background()
	if err != nil {
		log.Fatalf("Failed to initialize Cloudinary: %v", err)
	}

	file, err := os.Open(photo)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	uploadResult, err := cld.Upload.Upload(ctx, file, uploader.UploadParams{
		Folder:       "golang_uploads",
		ResourceType: "image",
	})
	if err != nil {
		log.Fatalf("Failed to upload image: %v", err)
	}

	return uploadResult.SecureURL
}

func Url(photo string) string {
	_ = godotenv.Load() // loads .env if exists

	cld, err := cloudinary.NewFromURL(os.Getenv("CLOUDINARY_URL"))
	if err != nil {
		log.Fatalf("Failed to initialize Cloudinary: %v", err)
	}
	image, err := cld.Image(photo)
	if err != nil {
		log.Fatalf("Failed to get image: %v", err)
	}
	imageUrl, err := image.String()
	if err != nil {
		log.Fatalf("Failed to get image URL: %v", err)
	}
	return imageUrl
}
