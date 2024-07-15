package internal

import (
	"bufio"
	"context"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Adapter struct {
	url string
}

func (a *S3Adapter) ObjectName() string {
	return "object"
}

func (a *S3Adapter) Scan(scanOpts ScanOpts) ([]ruleMatch, error) {
	return scanFiles(a, scanOpts)
}

func (a *S3Adapter) Init(url string) error {
	a.url = url
	return nil
}

func (a S3Adapter) FetchFiles() ([]string, error) {
	urlStr := a.url
	var files []string

	if strings.HasSuffix(urlStr, "/") {
		u, err := url.Parse(urlStr)
		if err != nil {
			return files, err
		}
		bucket := u.Host
		key := u.Path[1:]

		ctx := context.TODO()
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return files, err
		}

		svc := s3.NewFromConfig(cfg)

		params := &s3.ListObjectsInput{
			Bucket: aws.String(bucket),
			Prefix: aws.String(key),
		}

		resp, err := svc.ListObjects(ctx, params)
		if err != nil {
			return files, err
		}
		for _, key := range resp.Contents {
			files = append(files, "s3://"+bucket+"/"+*key.Key)
		}
	} else {
		files = append(files, urlStr)
	}

	return files, nil
}

func (a S3Adapter) FindFileMatches(filename string, matchFinder *MatchFinder) error {
	ctx := context.TODO()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}

	u, err := url.Parse(filename)
	if err != nil {
		return err
	}
	bucket := u.Host
	key := u.Path

	// TODO stream
	// TODO get file type before full download
	svc := s3.NewFromConfig(cfg)
	resp, err := svc.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key[1:]),
	})
	if err != nil {
		return err
	}

	return processFile(bufio.NewReader(resp.Body), matchFinder)
}
