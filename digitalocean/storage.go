package digitalocean

import (
	"bytes"
	"fmt"

	"github.com/algao1/imgrepo"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type ImageStorage struct {
	client *s3.S3
	bucket string
}

var _ imgrepo.ImageStorage = (*ImageStorage)(nil)

func NewImageStorage(key, secret, endpoint, region, bucket string) (*ImageStorage, error) {
	s3Config := &aws.Config{
		Credentials: credentials.NewStaticCredentials(key, secret, ""),
		Endpoint:    aws.String(endpoint),
		Region:      aws.String(region),
	}

	newSession, err := session.NewSession(s3Config)
	if err != nil {
		return nil, fmt.Errorf("%q: %w", "unable to create spaces session", err)
	}

	return &ImageStorage{client: s3.New(newSession), bucket: bucket}, nil
}

func (is *ImageStorage) Upload(img *imgrepo.Image) error {
	object := s3.PutObjectInput{
		Bucket: aws.String(is.bucket),
		Key:    aws.String(img.Id),
		Body:   bytes.NewReader(img.Raw),
		ACL:    aws.String("private"),
		Metadata: map[string]*string{
			"x-amz-meta-my-key": aws.String("your-value"), // required
		},
	}

	_, err := is.client.PutObject(&object)
	if err != nil {
		return fmt.Errorf("%q: %w", "unable to upload file", err)
	}

	return nil
}

func (is *ImageStorage) Download(id string) ([]byte, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(is.bucket),
		Key:    aws.String(id),
	}

	result, err := is.client.GetObject(input)
	if err != nil {
		return nil, fmt.Errorf("%q: %w", "unable to download file", err)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(result.Body)

	return buf.Bytes(), nil
}

func (is *ImageStorage) Delete(id string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(is.bucket),
		Key:    aws.String(id),
	}

	_, err := is.client.DeleteObject(input)
	if err != nil {
		return fmt.Errorf("%q: %w", "unable to delete file", err)
	}

	return nil
}
