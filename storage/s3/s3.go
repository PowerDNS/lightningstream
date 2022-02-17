package s3

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	s3config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"powerdns.com/platform/lightningstream/config"
	"powerdns.com/platform/lightningstream/storage"
)

const (
	// DefaultRegion is the default S3 region to use, if none is configured
	DefaultRegion = "us-east-1"
	// DefaultInitTimeout is the time we allow for initialisation, like credential
	// checking and bucket creation. We define this here, because we do not
	// pass a context when initialising a plugin.
	DefaultInitTimeout = 20 * time.Second
)

// Options describes the storage options for the S3 backend
type Options struct {
	// AccessKey and SecretKey are statically defined here.
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`

	// Region defaults to "us-east-1", which also works for Minio
	Region string `json:"region"`
	Bucket string `json:"bucket"`

	// EndpointURL can be set to something like "http://localhost:9000" when using Minio
	// instead of AWS S3.
	EndpointURL string `json:"endpoint_url"`

	// InitTimeout is the time we allow for initialisation, like credential
	// checking and bucket creation. It defaults to DefaultInitTimeout, which
	// is currently 20s.
	InitTimeout time.Duration `json:"init_timeout"`
}

func (o Options) Check() error {
	if o.AccessKey == "" {
		return fmt.Errorf("s3 storage.options: access_key is required")
	}
	if o.SecretKey == "" {
		return fmt.Errorf("s3 storage.options: secret_key is required")
	}
	if o.Bucket == "" {
		return fmt.Errorf("s3 storage.options: bucket is required")
	}
	return nil
}

type Backend struct {
	st       config.Storage
	opt      Options
	s3config aws.Config
	client   *s3.Client
}

func (b *Backend) List(ctx context.Context, prefix string) (storage.BlobList, error) {
	var blobs storage.BlobList

	paginator := s3.NewListObjectsV2Paginator(b.client, &s3.ListObjectsV2Input{
		Bucket:    aws.String(b.opt.Bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, obj := range page.Contents {
			if obj.Key == nil {
				continue // just in case, should not happen
			}
			blobs = append(blobs, storage.Blob{
				Name: *obj.Key,
				Size: obj.Size,
			})
		}
	}

	// Minio appears to return them sorted, but maybe not all implementations
	// will, so we sort explicitly.
	sort.Sort(blobs)

	return blobs, nil
}

func (b *Backend) Load(ctx context.Context, name string) ([]byte, error) {
	buf := manager.NewWriteAtBuffer(nil)
	downloader := manager.NewDownloader(b.client)
	_, err := downloader.Download(ctx, buf, &s3.GetObjectInput{
		Bucket: aws.String(b.opt.Bucket),
		Key:    aws.String(name),
	})
	if err != nil {
		if isResponseError(err, http.StatusNotFound) {
			return nil, os.ErrNotExist
		}
		return nil, err
	}
	return buf.Bytes(), nil
}

func (b *Backend) Store(ctx context.Context, name string, data []byte) error {
	uploader := manager.NewUploader(b.client)
	_, err := uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(b.opt.Bucket),
		Key:    aws.String(name),
		Body:   bytes.NewReader(data),
	})
	return err
}

func isResponseError(err error, statusCode int) bool {
	var responseError *awshttp.ResponseError
	if !errors.As(err, &responseError) {
		return false
	}
	return responseError.ResponseError.HTTPStatusCode() == statusCode
}

func New(st config.Storage) (*Backend, error) {
	// JSON roundtrip to get the options in a nice struct
	j, err := json.Marshal(st.Options)
	if err != nil {
		return nil, err
	}
	var opt Options
	if err := json.Unmarshal(j, &opt); err != nil {
		return nil, err
	}
	if opt.Region == "" {
		opt.Region = DefaultRegion
	}
	if opt.InitTimeout == 0 {
		opt.InitTimeout = DefaultInitTimeout
	}
	if err := opt.Check(); err != nil {
		return nil, err
	}

	// Some of the following calls require a context
	ctx, cancel := context.WithTimeout(context.TODO(), opt.InitTimeout)
	defer cancel()

	creds := credentials.NewStaticCredentialsProvider(opt.AccessKey, opt.SecretKey, "")
	cfg, err := s3config.LoadDefaultConfig(
		ctx,
		s3config.WithCredentialsProvider(creds),
		s3config.WithRegion(opt.Region))
	if err != nil {
		return nil, err
	}

	if opt.EndpointURL != "" {
		cfg.EndpointResolverWithOptions = aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               opt.EndpointURL,
					HostnameImmutable: true,
				}, nil
			},
		)
	}

	client := s3.NewFromConfig(cfg)

	// Create bucket if it does not exist
	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(opt.Bucket),
	})
	if err != nil && !isResponseError(err, http.StatusConflict) {
		return nil, err
	}

	b := &Backend{
		st:       st,
		opt:      opt,
		s3config: cfg,
		client:   client,
	}

	return b, nil
}

func init() {
	storage.RegisterBackend("s3", func(st config.Storage) (storage.Interface, error) {
		return New(st)
	})
}
