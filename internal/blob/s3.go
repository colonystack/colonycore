package blob

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	aws "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3 implements BlobStore using an S3-compatible backend (AWS S3 or MinIO).
// Minimal surface area: single bucket. Keys map to object keys directly.
type S3 struct {
	client  *s3.Client
	bucket  string
	presign *s3.PresignClient
	baseURL *url.URL // optional explicit endpoint base for constructing local-style URLs
}

// S3Config holds explicit construction parameters (mostly for tests). For prod
// we rely primarily on environment variables.
type S3Config struct {
	Region          string
	Bucket          string
	Endpoint        string // optional; if set enables custom endpoint (e.g. MinIO)
	AccessKeyID     string // optional (falls back to default credentials chain)
	SecretAccessKey string // optional
	SessionToken    string // optional
	PathStyle       bool
}

// Environment variables (documented in README):
//   COLONYCORE_BLOB_DRIVER=s3
//   COLONYCORE_BLOB_S3_BUCKET=<bucket> (required)
//   COLONYCORE_BLOB_S3_REGION=<region> (default us-east-1)
//   COLONYCORE_BLOB_S3_ENDPOINT=<url> (optional, for MinIO)
//   COLONYCORE_BLOB_S3_PATH_STYLE=true|false (default false)
//   AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY / AWS_SESSION_TOKEN (optional)

// NewS3 creates an S3 blob store from S3Config.
func NewS3(ctx context.Context, cfg S3Config) (*S3, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("s3 bucket required")
	}
	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}
	// Build AWS config
	var loadOpts []func(*config.LoadOptions) error
	if region != "" {
		loadOpts = append(loadOpts, config.WithRegion(region))
	}
	// Custom endpoint handled later via s3.Options below to avoid deprecated resolver path.
	awsCfg, err := config.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, err
	}
	// (Optional explicit static credentials path removed; rely on default chain to avoid extra indirection.)
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.PathStyle {
			o.UsePathStyle = true
		}
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
	})
	ps := s3.NewPresignClient(client)
	var base *url.URL
	if cfg.Endpoint != "" {
		if u, err := url.Parse(cfg.Endpoint); err == nil {
			base = u
		}
	}
	return &S3{client: client, bucket: cfg.Bucket, presign: ps, baseURL: base}, nil
}

func (s *S3) Driver() Driver { return DriverS3 }

func (s *S3) Put(ctx context.Context, key string, r io.Reader, opts PutOptions) (Info, error) {
	input := &s3.PutObjectInput{Bucket: &s.bucket, Key: &key, Body: r}
	if opts.ContentType != "" {
		input.ContentType = &opts.ContentType
	}
	if len(opts.Metadata) > 0 {
		input.Metadata = opts.Metadata
	}
	// Emulate create-only via Head first.
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &s.bucket, Key: &key})
	if err == nil {
		return Info{}, fmt.Errorf("blob %s already exists", key)
	}
	if _, err := s.client.PutObject(ctx, input); err != nil {
		return Info{}, err
	}
	return s.Head(ctx, key)
}

func (s *S3) Get(ctx context.Context, key string) (Info, io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: &s.bucket, Key: &key})
	if err != nil {
		return Info{}, nil, err
	}
	size := int64(0)
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	info := s.fromHead(key, size, out.ContentType, out.ETag, out.Metadata, out.LastModified)
	return info, out.Body, nil
}

func (s *S3) Head(ctx context.Context, key string) (Info, error) {
	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &s.bucket, Key: &key})
	if err != nil {
		return Info{}, err
	}
	size := int64(0)
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	return s.fromHead(key, size, out.ContentType, out.ETag, out.Metadata, out.LastModified), nil
}

func (s *S3) Delete(ctx context.Context, key string) (bool, error) {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &s.bucket, Key: &key})
	if err != nil {
		return false, err
	}
	// Head to confirm existence pre-delete is extra round trip; for simplicity assume existed if no error.
	return true, nil
}

func (s *S3) List(ctx context.Context, prefix string) ([]Info, error) {
	var infos []Info
	var token *string
	for {
		out, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: &s.bucket, Prefix: &prefix, ContinuationToken: token})
		if err != nil {
			return nil, err
		}
		for _, obj := range out.Contents {
			var size int64
			if obj.Size != nil {
				size = *obj.Size
			}
			infos = append(infos, Info{Key: aws.ToString(obj.Key), Size: size, LastModified: aws.ToTime(obj.LastModified)})
		}
		if out.IsTruncated != nil && *out.IsTruncated && out.NextContinuationToken != nil {
			token = out.NextContinuationToken
			continue
		}
		break
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].Key < infos[j].Key })
	return infos, nil
}

func (s *S3) PresignURL(ctx context.Context, key string, opts SignedURLOptions) (string, error) {
	method := strings.ToUpper(opts.Method)
	if method == "" {
		method = "GET"
	}
	if method != "GET" {
		return "", ErrUnsupported
	}
	expiry := opts.Expiry
	if expiry <= 0 {
		expiry = 15 * time.Minute
	}
	pout, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: &s.bucket, Key: &key}, func(po *s3.PresignOptions) { po.Expires = expiry })
	if err != nil {
		return "", err
	}
	return pout.URL, nil
}

func (s *S3) fromHead(key string, size int64, contentType *string, etag *string, md map[string]string, lastModified *time.Time) Info {
	var ct, et string
	if contentType != nil {
		ct = *contentType
	}
	if etag != nil {
		et = strings.Trim(*etag, "\"")
	}
	lm := time.Now().UTC()
	if lastModified != nil {
		lm = *lastModified
	}
	return Info{Key: key, Size: size, ContentType: ct, ETag: et, Metadata: md, LastModified: lm}
}

// --- Factory from environment ---

// OpenFromEnv constructs an S3 store from process environment.
func OpenFromEnv(ctx context.Context) (*S3, error) {
	bucket := os.Getenv("COLONYCORE_BLOB_S3_BUCKET")
	if bucket == "" {
		return nil, fmt.Errorf("COLONYCORE_BLOB_S3_BUCKET required for s3 driver")
	}
	cfg := S3Config{
		Bucket:    bucket,
		Region:    os.Getenv("COLONYCORE_BLOB_S3_REGION"),
		Endpoint:  os.Getenv("COLONYCORE_BLOB_S3_ENDPOINT"),
		PathStyle: strings.EqualFold(os.Getenv("COLONYCORE_BLOB_S3_PATH_STYLE"), "true"),
	}
	return NewS3(ctx, cfg)
}
