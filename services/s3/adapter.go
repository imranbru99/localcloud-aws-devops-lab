package s3

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/imrandev/localcloud/gateway/pkg/context"
	"github.com/imrandev/localcloud/gateway/pkg/errors"
	"github.com/imrandev/localcloud/services"
)

type S3Adapter struct {
	mu      sync.RWMutex
	buckets map[string]*BucketMeta
	dataDir string
}

type BucketMeta struct {
	Name         string            `json:"name"`
	CreationDate time.Time         `json:"creation_date"`
	Versioning   string            `json:"versioning"` // "Enabled" or "Suspended"
	Policy       string            `json:"policy,omitempty"`
	Cors         string            `json:"cors,omitempty"`
	Tags         map[string]string `json:"tags"`
}

var DefaultS3 *S3Adapter

func init() {
	dataDir := os.Getenv("LOCALCLOUD_DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
	s3Dir := filepath.Join(dataDir, "s3")
	_ = os.MkdirAll(s3Dir, 0755)

	DefaultS3 = &S3Adapter{
		buckets: make(map[string]*BucketMeta),
		dataDir: s3Dir,
	}
	services.Register(DefaultS3)
}

func GetBuckets() []*BucketMeta {
	if DefaultS3 == nil {
		return []*BucketMeta{}
	}
	DefaultS3.mu.RLock()
	defer DefaultS3.mu.RUnlock()
	var list []*BucketMeta
	for _, b := range DefaultS3.buckets {
		list = append(list, b)
	}
	return list
}

func (a *S3Adapter) Name() string {
	return "s3"
}

func (a *S3Adapter) Protocol() services.Protocol {
	return services.ProtocolRestXML
}

func (a *S3Adapter) Operations() map[string]services.Handler {
	return map[string]services.Handler{
		"CreateBucket":        a.CreateBucket,
		"ListBuckets":         a.ListBuckets,
		"DeleteBucket":        a.DeleteBucket,
		"HeadBucket":          a.HeadBucket,
		"PutObject":           a.PutObject,
		"GetObject":           a.GetObject,
		"DeleteObject":        a.DeleteObject,
		"ListObjects":         a.ListObjectsV2,
		"ListObjectsV2":       a.ListObjectsV2,
		"HeadObject":          a.HeadObject,
		"PutBucketPolicy":     a.PutBucketPolicy,
		"GetBucketPolicy":     a.GetBucketPolicy,
		"DeleteBucketPolicy":  a.DeleteBucketPolicy,
		"PutBucketVersioning": a.PutBucketVersioning,
		"GetBucketVersioning": a.GetBucketVersioning,
	}
}

func (a *S3Adapter) CreateBucket(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	bucket := ctx.ParsedParams["Bucket"]
	if bucket == "" {
		return nil, errors.NewAWSError(http.StatusBadRequest, "InvalidBucketName", "Bucket name cannot be empty")
	}

	bucketPath := filepath.Join(a.dataDir, bucket)
	if err := os.MkdirAll(bucketPath, 0755); err != nil {
		return nil, errors.NewAWSError(http.StatusInternalServerError, "InternalFailure", err.Error())
	}

	a.buckets[bucket] = &BucketMeta{
		Name:         bucket,
		CreationDate: time.Now().UTC(),
		Tags:         make(map[string]string),
	}

	return nil, nil
}

type ListAllMyBucketsResult struct {
	XMLName xml.Name `xml:"ListAllMyBucketsResult"`
	Xmlns   string   `xml:"xmlns,attr"`
	Owner   Owner    `xml:"Owner"`
	Buckets struct {
		Bucket []BucketEntry `xml:"Bucket"`
	} `xml:"Buckets"`
}

type Owner struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName"`
}

type BucketEntry struct {
	Name         string `xml:"Name"`
	CreationDate string `xml:"CreationDate"`
}

func (a *S3Adapter) ListBuckets(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	res := ListAllMyBucketsResult{
		Xmlns: "http://s3.amazonaws.com/doc/2006-03-01/",
		Owner: Owner{
			ID:          "localcloud-owner-" + ctx.AccountID,
			DisplayName: "root",
		},
	}

	for _, b := range a.buckets {
		res.Buckets.Bucket = append(res.Buckets.Bucket, BucketEntry{
			Name:         b.Name,
			CreationDate: b.CreationDate.Format(time.RFC3339),
		})
	}

	return res, nil
}

func (a *S3Adapter) DeleteBucket(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	bucket := ctx.ParsedParams["Bucket"]
	delete(a.buckets, bucket)
	_ = os.RemoveAll(filepath.Join(a.dataDir, bucket))
	return nil, nil
}

func (a *S3Adapter) HeadBucket(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	bucket := ctx.ParsedParams["Bucket"]
	if _, ok := a.buckets[bucket]; !ok {
		return nil, errors.ErrNoSuchBucket
	}
	return nil, nil
}

func (a *S3Adapter) PutObject(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	bucket := ctx.ParsedParams["Bucket"]
	key := ctx.ParsedParams["Key"]

	if bucket == "" || key == "" {
		return nil, errors.ErrInvalidParameter
	}

	objPath := filepath.Join(a.dataDir, bucket, key)
	if err := os.MkdirAll(filepath.Dir(objPath), 0755); err != nil {
		return nil, errors.NewAWSError(http.StatusInternalServerError, "InternalFailure", err.Error())
	}

	f, err := os.Create(objPath)
	if err != nil {
		return nil, errors.NewAWSError(http.StatusInternalServerError, "InternalFailure", err.Error())
	}
	defer f.Close()

	hasher := md5.New()
	writer := io.MultiWriter(f, hasher)
	_, _ = writer.Write(ctx.RawBody)

	var _ = "\"" + hex.EncodeToString(hasher.Sum(nil)) + "\""
	return nil, nil
}

func (a *S3Adapter) GetObject(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	bucket := ctx.ParsedParams["Bucket"]
	key := ctx.ParsedParams["Key"]

	objPath := filepath.Join(a.dataDir, bucket, key)
	content, err := os.ReadFile(objPath)
	if err != nil {
		return nil, errors.ErrNoSuchKey
	}

	return content, nil
}

func (a *S3Adapter) DeleteObject(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	bucket := ctx.ParsedParams["Bucket"]
	key := ctx.ParsedParams["Key"]

	objPath := filepath.Join(a.dataDir, bucket, key)
	_ = os.Remove(objPath)
	return nil, nil
}

type ListBucketResult struct {
	XMLName     xml.Name   `xml:"ListBucketResult"`
	Xmlns       string     `xml:"xmlns,attr"`
	Name        string     `xml:"Name"`
	Prefix      string     `xml:"Prefix"`
	KeyCount    int        `xml:"KeyCount"`
	MaxKeys     int        `xml:"MaxKeys"`
	IsTruncated bool       `xml:"IsTruncated"`
	Contents    []S3Object `xml:"Contents"`
}

type S3Object struct {
	Key          string `xml:"Key"`
	LastModified string `xml:"LastModified"`
	ETag         string `xml:"ETag"`
	Size         int64  `xml:"Size"`
	StorageClass string `xml:"StorageClass"`
}

func (a *S3Adapter) ListObjectsV2(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	bucket := ctx.ParsedParams["Bucket"]
	prefix := ctx.ParsedParams["prefix"]

	bucketPath := filepath.Join(a.dataDir, bucket)
	res := ListBucketResult{
		Xmlns:       "http://s3.amazonaws.com/doc/2006-03-01/",
		Name:        bucket,
		Prefix:      prefix,
		MaxKeys:     1000,
		IsTruncated: false,
		Contents:    make([]S3Object, 0),
	}

	_ = filepath.Walk(bucketPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(bucketPath, path)
		if prefix == "" || strings.HasPrefix(rel, prefix) {
			res.Contents = append(res.Contents, S3Object{
				Key:          rel,
				LastModified: info.ModTime().UTC().Format(time.RFC3339),
				ETag:         "\"d41d8cd98f00b204e9800998ecf8427e\"",
				Size:         info.Size(),
				StorageClass: "STANDARD",
			})
		}
		return nil
	})

	res.KeyCount = len(res.Contents)
	return res, nil
}

func (a *S3Adapter) HeadObject(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	bucket := ctx.ParsedParams["Bucket"]
	key := ctx.ParsedParams["Key"]

	objPath := filepath.Join(a.dataDir, bucket, key)
	if _, err := os.Stat(objPath); err != nil {
		return nil, errors.ErrNoSuchKey
	}
	return nil, nil
}

func (a *S3Adapter) PutBucketPolicy(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	bucket := ctx.ParsedParams["Bucket"]
	if b, ok := a.buckets[bucket]; ok {
		b.Policy = string(ctx.RawBody)
	}
	return nil, nil
}

func (a *S3Adapter) GetBucketPolicy(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	bucket := ctx.ParsedParams["Bucket"]
	if b, ok := a.buckets[bucket]; ok && b.Policy != "" {
		return []byte(b.Policy), nil
	}
	return nil, errors.NewAWSError(http.StatusNotFound, "NoSuchBucketPolicy", "The bucket policy does not exist")
}

func (a *S3Adapter) DeleteBucketPolicy(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	bucket := ctx.ParsedParams["Bucket"]
	if b, ok := a.buckets[bucket]; ok {
		b.Policy = ""
	}
	return nil, nil
}

type GetBucketVersioningResult struct {
	XMLName xml.Name `xml:"VersioningConfiguration"`
	Xmlns   string   `xml:"xmlns,attr"`
	Status  string   `xml:"Status,omitempty"`
}

func (a *S3Adapter) PutBucketVersioning(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	bucket := ctx.ParsedParams["Bucket"]
	if b, ok := a.buckets[bucket]; ok {
		b.Versioning = "Enabled"
	}
	return nil, nil
}

func (a *S3Adapter) GetBucketVersioning(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	bucket := ctx.ParsedParams["Bucket"]
	status := ""
	if b, ok := a.buckets[bucket]; ok {
		status = b.Versioning
	}
	return GetBucketVersioningResult{
		Xmlns:  "http://s3.amazonaws.com/doc/2006-03-01/",
		Status: status,
	}, nil
}

func (a *S3Adapter) ResourceTypes() []services.ResourceType {
	return []services.ResourceType{
		{Name: "Bucket", ARNFormat: "arn:aws:s3:::{bucket}", Tags: true, Columns: []string{"Name", "CreationDate", "Versioning"}},
		{Name: "Object", ARNFormat: "arn:aws:s3:::{bucket}/{key}", Tags: true, Columns: []string{"Key", "Size", "LastModified", "StorageClass"}},
	}
}

func (a *S3Adapter) IAMActions() []services.ActionDef {
	return []services.ActionDef{
		{Name: "s3:CreateBucket", ResourceARNs: []string{"arn:aws:s3:::*"}},
		{Name: "s3:ListBuckets", ResourceARNs: []string{"arn:aws:s3:::*"}},
		{Name: "s3:DeleteBucket", ResourceARNs: []string{"arn:aws:s3:::*"}},
		{Name: "s3:PutObject", ResourceARNs: []string{"arn:aws:s3:::*/*"}},
		{Name: "s3:GetObject", ResourceARNs: []string{"arn:aws:s3:::*/*"}},
		{Name: "s3:DeleteObject", ResourceARNs: []string{"arn:aws:s3:::*/*"}},
		{Name: "s3:ListBucket", ResourceARNs: []string{"arn:aws:s3:::*"}},
	}
}

func (a *S3Adapter) HealthCheck() services.Health {
	return services.Health{Status: "healthy"}
}

func (a *S3Adapter) EmitEvents() []services.EventDef {
	return []services.EventDef{
		{Name: "CreateBucket", Source: "s3.amazonaws.com"},
		{Name: "PutObject", Source: "s3.amazonaws.com"},
		{Name: "DeleteObject", Source: "s3.amazonaws.com"},
	}
}

func (a *S3Adapter) Metrics() []services.MetricDef {
	return []services.MetricDef{
		{Name: "BucketSizeBytes", Namespace: "AWS/S3", Unit: "Bytes"},
		{Name: "NumberOfObjects", Namespace: "AWS/S3", Unit: "Count"},
	}
}

func (a *S3Adapter) Quotas() []services.Quota {
	return []services.Quota{
		{Name: "BucketsPerAccount", DefaultValue: 100, Description: "Maximum number of S3 buckets per account"},
	}
}

func (a *S3Adapter) Reset(account, region string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.buckets = make(map[string]*BucketMeta)
	_ = os.RemoveAll(a.dataDir)
	_ = os.MkdirAll(a.dataDir, 0755)
	return nil
}

func (a *S3Adapter) Snapshot() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (a *S3Adapter) Restore(data map[string]interface{}) error {
	return nil
}
