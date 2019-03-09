package internal

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func findS3Files(urlStr string) []string {
	var files []string

	if strings.HasSuffix(urlStr, "/") {
		u, err1 := url.Parse(urlStr)
		if err1 != nil {
			log.Fatal(err1)
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

	return files
}

func downloadS3File(filename string) ReadSeekCloser {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	u, err := url.Parse(filename)
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}

	buff, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return bytes.NewReader(buff)
}
