package internal

import (
	"bufio"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type S3Adapter struct {
	url string
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

		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))

		svc := s3.New(sess)

		params := &s3.ListObjectsInput{
			Bucket: aws.String(bucket),
			Prefix: aws.String(key),
		}

		resp, _ := svc.ListObjects(params)
		for _, key := range resp.Contents {
			files = append(files, "s3://"+bucket+"/"+*key.Key)
		}
	} else {
		files = append(files, urlStr)
	}

	return files, nil
}

func (a S3Adapter) FindFileMatches(filename string) ([][]string, int, error) {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	u, err := url.Parse(filename)
	if err != nil {
		return nil, 0, err
	}
	bucket := u.Host
	key := u.Path

	// TODO stream
	// TODO get file type before full download
	svc := s3.New(sess)
	resp, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, 0, err
	}

	return processFile(bufio.NewReader(resp.Body))
}
