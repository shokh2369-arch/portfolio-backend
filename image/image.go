package image

import (
	"fmt"
	"log"
	"os"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/joho/godotenv"
)

func BuildURL(publicId string) string {
	dataURL := os.Getenv("CLOUDINARY_URL")
	err := godotenv.Load(dataURL)

	if err != nil {
		log.Fatal("Error loading .env file")
	}

	cld, err := cloudinary.New()
	if err != nil {
		log.Fatal("cloudinary init error:", err)
	}
	fmt.Scanln(&publicId)

	img, err := cld.Image(publicId)
	if err != nil {
		log.Fatal("image error:", err)
	}

	url, err := img.String()
	if err != nil {
		log.Fatal("url build error:", err)
	}

	return url
}
