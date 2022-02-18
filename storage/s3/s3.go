package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	s3config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gopkg.in/yaml.v2"

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
	// UpdateMarkerFilename is the filename used for the update marker functionality
	UpdateMarkerFilename = "update-marker"
	// DefaultUpdateMarkerForceListInterval is the default value for
	// UpdateMarkerForceListInterval.
	DefaultUpdateMarkerForceListInterval = 5 * time.Minute
)

// Options describes the storage options for the S3 backend
type Options struct {
	// AccessKey and SecretKey are statically defined here.
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`

	// Region defaults to "us-east-1", which also works for Minio
	Region string `yaml:"region"`
	Bucket string `yaml:"bucket"`
	// CreateBucket tells us to try to create the bucket
	CreateBucket bool `yaml:"create_bucket"`

	// EndpointURL can be set to something like "http://localhost:9000" when using Minio
	// instead of AWS S3.
	EndpointURL string `yaml:"endpoint_url"`

	// InitTimeout is the time we allow for initialisation, like credential
	// checking and bucket creation. It defaults to DefaultInitTimeout, which
	// is currently 20s.
	InitTimeout time.Duration `yaml:"init_timeout"`

	// UseUpdateMarker makes the backend write and read a file to determine if
	// it can cache the last List command. The file contains the name of the
	// last file stored.
	// This can reduce the number of LIST commands sent to S3, replacing them
	// with GET commands that are about 12x cheaper.
	// If enabled, it MUST be enabled on all instances!
	// CAVEAT: This will NOT work correctly if the bucket itself is replicated
	//         in an active-active fashion between data centers! In that case
	//         do not enable this option.
	UseUpdateMarker bool `yaml:"use_update_marker"`
	// UpdateMarkerForceListInterval is used when UseUpdateMarker is enabled.
	// A LIST command will be sent when this interval has passed without a
	// change in marker, to ensure a full sync even if the marker would for
	// some reason get out of sync.
	UpdateMarkerForceListInterval time.Duration `yaml:"update_marker_force_list_interval"`
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

	mu         sync.Mutex
	lastMarker string
	lastList   storage.BlobList
	lastTime   time.Time
}

func (b *Backend) List(ctx context.Context, prefix string) (storage.BlobList, error) {
	if !b.opt.UseUpdateMarker {
		return b.doList(ctx, prefix)
	}

	// Request and cache full list, and use marker file to invalidate the cache
	now := time.Now()
	data, err := b.Load(ctx, UpdateMarkerFilename)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	current := string(data)

	b.mu.Lock()
	lastMarker := b.lastMarker
	age := now.Sub(b.lastTime)
	b.mu.Unlock()

	var blobs storage.BlobList
	if current != lastMarker || age >= b.opt.UpdateMarkerForceListInterval {
		// Update cache
		blobs, err = b.doList(ctx, "") // all, no prefix
		if err != nil {
			return nil, err
		}

		b.mu.Lock()
		b.lastMarker = current
		b.lastList = blobs
		b.mu.Unlock()
	} else {
		b.mu.Lock()
		blobs = b.lastList
		b.mu.Unlock()
	}
	return blobs.WithPrefix(prefix), nil
}

func (b *Backend) doList(ctx context.Context, prefix string) (storage.BlobList, error) {
	var blobs storage.BlobList

	paginator := s3.NewListObjectsV2Paginator(b.client, &s3.ListObjectsV2Input{
		Bucket:    aws.String(b.opt.Bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	})

	for paginator.HasMorePages() {
		metricCalls.WithLabelValues("list").Inc()
		metricLastCallTimestamp.WithLabelValues("list").SetToCurrentTime()
		page, err := paginator.NextPage(ctx)
		if err != nil {
			metricCallErrors.WithLabelValues("list").Inc()
			return nil, err
		}
		for _, obj := range page.Contents {
			if obj.Key == nil {
				continue // just in case, should not happen
			}
			if *obj.Key == UpdateMarkerFilename {
				continue
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
	metricCalls.WithLabelValues("load").Inc()
	metricLastCallTimestamp.WithLabelValues("load").SetToCurrentTime()
	_, err := downloader.Download(ctx, buf, &s3.GetObjectInput{
		Bucket: aws.String(b.opt.Bucket),
		Key:    aws.String(name),
	})
	if err != nil {
		metricCallErrors.WithLabelValues("load").Inc()
		if isResponseError(err, http.StatusNotFound) {
			return nil, os.ErrNotExist
		}
		return nil, err
	}
	return buf.Bytes(), nil
}

func (b *Backend) Store(ctx context.Context, name string, data []byte) error {
	if err := b.doStore(ctx, name, data); err != nil {
		return err
	}
	if b.opt.UseUpdateMarker {
		if err := b.doStore(ctx, UpdateMarkerFilename, []byte(name)); err != nil {
			return err
		}
	}
	return nil
}

func (b *Backend) doStore(ctx context.Context, name string, data []byte) error {
	metricCalls.WithLabelValues("store").Inc()
	metricLastCallTimestamp.WithLabelValues("store").SetToCurrentTime()
	uploader := manager.NewUploader(b.client)
	_, err := uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(b.opt.Bucket),
		Key:    aws.String(name),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		metricCallErrors.WithLabelValues("store").Inc()
	}
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
	y, err := yaml.Marshal(st.Options)
	if err != nil {
		return nil, err
	}
	var opt Options
	if err := yaml.UnmarshalStrict(y, &opt); err != nil {
		return nil, err
	}
	if opt.Region == "" {
		opt.Region = DefaultRegion
	}
	if opt.InitTimeout == 0 {
		opt.InitTimeout = DefaultInitTimeout
	}
	if opt.UpdateMarkerForceListInterval == 0 {
		opt.UpdateMarkerForceListInterval = DefaultUpdateMarkerForceListInterval
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

	if opt.CreateBucket {
		// Create bucket if it does not exist
		metricCalls.WithLabelValues("create-bucket").Inc()
		metricLastCallTimestamp.WithLabelValues("create-bucket").SetToCurrentTime()
		_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(opt.Bucket),
		})
		if err != nil && !isResponseError(err, http.StatusConflict) {
			metricCallErrors.WithLabelValues("create-bucket").Inc()
			return nil, err
		}
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
