package s3

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"powerdns.com/platform/lightningstream/config"
	"powerdns.com/platform/lightningstream/storage/tester"
)

// TestConfigPathEnv is the path to a YAML file with the Options
// with an S3 bucket configuration that can be used for testing.
// This is intentionally different from the normal YAML config, to prevent mistakes.
// The bucket will be emptied before every run!!!
//
// To run a Minio for this :
//
//     env MINIO_ROOT_USER=test MINIO_ROOT_PASSWORD=secret minio server /tmp/test-data/
//
// Example test config:
//
//     {
//       "access_key": "test",
//       "secret_key": "verysecret",
//       "region": "us-east-1",
//       "bucket": "test-lightningstream",
//       "endpoint_url": "http://127.0.0.1:9000"
//     }
//
const TestConfigPathEnv = "LIGHTNINGSTREAM_TEST_S3_CONFIG"

func getBackend(ctx context.Context, t *testing.T) (b *Backend) {
	cfgPath := os.Getenv(TestConfigPathEnv)
	if cfgPath == "" {
		t.Skipf("S3 tests skipped, set the %s env var to run these", TestConfigPathEnv)
		return
	}

	cfgContents, err := ioutil.ReadFile(cfgPath)
	require.NoError(t, err)

	var d map[string]interface{}
	err = yaml.Unmarshal(cfgContents, &d)
	require.NoError(t, err)

	cfg := config.Storage{
		Type:    "s3",
		Options: d,
	}

	b, err = New(cfg)
	require.NoError(t, err)

	cleanup := func() {
		blobs, err := b.doList(ctx, "")
		if err != nil {
			t.Logf("Blobs list error: %s", err)
			return
		}
		for _, blob := range blobs {
			_, err := b.client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(b.opt.Bucket),
				Key:    aws.String(blob.Name),
			})
			if err != nil {
				t.Logf("Object delete error: %s", err)
			}
		}
		// This one is not returned by the List command
		_, _ = b.client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(b.opt.Bucket),
			Key:    aws.String(UpdateMarkerFilename),
		})
	}
	t.Cleanup(cleanup)
	cleanup()

	return b
}

func TestBackend(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	b := getBackend(ctx, t)
	tester.DoBackendTests(t, b)
	assert.Equal(t, "", b.lastMarker)
}

func TestBackend_marker(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	b := getBackend(ctx, t)
	b.opt.UseUpdateMarker = true

	tester.DoBackendTests(t, b)
	assert.Equal(t, "bar-1", b.lastMarker)

	data, err := b.Load(ctx, UpdateMarkerFilename)
	assert.NoError(t, err)
	assert.Equal(t, "bar-1", string(data))
}
