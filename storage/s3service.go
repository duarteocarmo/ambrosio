package storage

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path"
	"time"

	_ "image/jpeg"
	_ "image/png"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/chai2010/webp"
)

const (
	BucketName   = "photos"
	BucketRegion = "auto"
)

type Photo struct {
	Url      string
	ID       string
	Date     string
	Caption  *string
	Location *string
}

type ImageBytes struct {
	Original  []byte
	Thumbnail []byte
}

type SubImager interface {
	SubImage(r image.Rectangle) image.Image
}

func getS3Client() *s3.Client {
	accessKeyId := os.Getenv("AWS_ACCESS_KEY_ID")
	accessKeySecret := os.Getenv("AWS_SECRET_ACCESS_KEY")
	bucketUrl := os.Getenv("BUCKET_URL")

	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:           bucketUrl,
			SigningRegion: BucketRegion,
		}, nil
	})
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(r2Resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyId, accessKeySecret, "")),
	)
	if err != nil {
		log.Fatal(err)
	}

	client := s3.NewFromConfig(cfg)
	return client
}

func (p *Photo) Create() error {

	currentTime := time.Now()
	p.Date = currentTime.Format("2006-01-02 15:04:05")

	hasher := sha1.New()
	hasher.Write([]byte(p.Date))
	p.ID = fmt.Sprintf("%x", hasher.Sum(nil))

	pBytes, err := processPhoto(p)
	if err != nil {
		return err
	}

	client := getS3Client()

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(BucketName),
		Key:    aws.String(path.Base(p.ID) + ".png"),
		Body:   bytes.NewReader(pBytes.Original),
	})
	if err != nil {
		return err
	}

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(BucketName),
		Key:    aws.String(path.Base(p.ID) + ".webp"),
		Body:   bytes.NewReader(pBytes.Thumbnail),
	})
	if err != nil {
		return err
	}

	jsonBytes, err := json.Marshal(struct {
		Caption  *string `json:"caption,omitempty"`
		Location *string `json:"location,omitempty"`
		Date     string  `json:"date"`
		ID       string  `json:"id"`
	}{
		Caption:  p.Caption,
		Location: p.Location,
		Date:     p.Date,
		ID:       p.ID,
	})
	if err != nil {
		return err
	}

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(BucketName),
		Key:    aws.String(path.Base(p.ID) + ".json"),
		Body:   bytes.NewReader(jsonBytes),
	})
	if err != nil {
		return err
	}

	return nil

}

func processPhoto(p *Photo) (ImageBytes, error) {
	resp, err := http.Get(p.Url)
	if err != nil {
		return ImageBytes{}, fmt.Errorf("failed to get photo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ImageBytes{}, fmt.Errorf("unexpected status code: %s", resp.Status)
	}

	photoBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return ImageBytes{}, fmt.Errorf("failed to read photo body: %w", err)
	}

	img, _, err := image.Decode(bytes.NewReader(photoBytes))
	if err != nil {
		return ImageBytes{}, fmt.Errorf("failed to decode photo: %w", err)
	}

	// generate square thumbnail
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	squareSide := int(math.Min(float64(width), float64(height)))

	startX := (width - squareSide) / 2
	startY := (height - squareSide) / 2
	endX := startX + squareSide
	endY := startY + squareSide

	cropSize := image.Rect(startX, startY, endX, endY)
	croppedImage, ok := img.(interface {
		SubImage(r image.Rectangle) image.Image
	})
	if !ok {
		log.Fatal("Image does not support sub-imaging")
	}
	croppedImg := croppedImage.SubImage(cropSize)

	var webpBytes bytes.Buffer
	if err := webp.Encode(&webpBytes, croppedImg, &webp.Options{Lossless: false, Quality: 50, Exact: false}); err != nil {
		return ImageBytes{}, fmt.Errorf("failed to encode photo to WebP: %w", err)
	}

	imageData := ImageBytes{
		Original:  photoBytes,
		Thumbnail: webpBytes.Bytes(),
	}

	log.Println("Successfully converted to WebP format")

	return imageData, nil

}
