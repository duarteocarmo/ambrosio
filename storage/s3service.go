package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"io"
	"log"
	"net/http"
	"os"
	"path"
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

	resp, err := http.Get(p.Url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("received non-200 response code")
	}
	photoBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fileKey := path.Base(p.ID) + ".png"
	jsonKey := fileKey + ".json"
	client := getS3Client()

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(BucketName),
		Key:    aws.String(fileKey),
		Body:   bytes.NewReader(photoBytes),
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
		Key:    aws.String(jsonKey),
		Body:   bytes.NewReader(jsonBytes),
	})
	if err != nil {
		return err
	}

	return nil
}
