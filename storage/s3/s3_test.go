package s3

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"powerdns.com/platform/lightningstream/config"
	"powerdns.com/platform/lightningstream/storage/tester"
)

// TestConfigPathEnv is the path to a JSON file with the Options
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

func TestBackend(t *testing.T) {
	cfgPath := os.Getenv(TestConfigPathEnv)
	if cfgPath == "" {
		t.Skipf("S3 tests skipped, set the %s env var to run these", TestConfigPathEnv)
		return
	}

	cfgContents, err := ioutil.ReadFile(cfgPath)
	assert.NoError(t, err)

	var d map[string]interface{}
	err = json.Unmarshal(cfgContents, &d)
	assert.NoError(t, err)

	cfg := config.Storage{
		Type:    "s3",
		Options: d,
	}

	b, err := New(cfg)
	assert.NoError(t, err)

	ctx := context.TODO()

	cleanup := func() {
		blobs, err := b.List(ctx, "")
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

		_, err = b.client.DeleteBucket(ctx, &s3.DeleteBucketInput{
			Bucket: aws.String(b.opt.Bucket),
		})
		if err != nil {
			t.Logf("Bucket delete error: %s", err)
		}
	}
	t.Cleanup(cleanup)
	cleanup()

	_, err = b.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(b.opt.Bucket),
	})
	assert.NoError(t, err)

	tester.DoBackendTests(t, b)
}
