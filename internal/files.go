package internal

import (
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/h2non/filetype"

	pdfcontent "github.com/unidoc/unidoc/pdf/contentstream"
	pdf "github.com/unidoc/unidoc/pdf/model"
)

type ReadSeekCloser interface {
	io.Reader
	io.Seeker
}

// TODO skip certain file types
func findFiles(urlStr string) []string {
	var files []string

	if strings.HasPrefix(urlStr, "file://") {
		root := urlStr[7:]
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err == nil {
				files = append(files, path)
			}
			return nil
		})

		if err != nil {
			log.Fatal(err)
		}
	} else {
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
	}

	return files
}

func findScannerMatches(scanner *bufio.Scanner) ([][]string, int) {
	matchedValues := make([][]string, len(regexRules)+1)
	nameIndex := len(regexRules)
	count := 0

	for scanner.Scan() {
		v := scanner.Text()
		count += 1

		for i, rule := range regexRules {
			if rule.Regex.MatchString(v) {
				matchedValues[i] = append(matchedValues[i], v)
			}
		}

		tokens := tokenizer.Split(strings.ToLower(v), -1)
		if anyMatches(tokens) {
			matchedValues[nameIndex] = append(matchedValues[nameIndex], v)
		}
	}

	return matchedValues, count
}

// TODO read metadata for certain file types
func findFileMatches(filename string) ([][]string, int) {
	var file ReadSeekCloser

	if strings.HasPrefix(filename, "s3://") {
		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))

		u, err1 := url.Parse(filename)
		if err1 != nil {
			log.Fatal(err1)
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

		buff, err2 := ioutil.ReadAll(resp.Body)
		if err2 != nil {
			log.Fatal(err)
		}

		file = bytes.NewReader(buff)
	} else {
		f, err := os.Open(filename)
		if err != nil {
			log.Fatal(err)
		}

		defer f.Close()

		file = f
	}

	// we only have to pass the file header = first 261 bytes
	head := make([]byte, 261)
	file.Read(head)
	kind, err := filetype.Match(head)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println(kind.MIME.Value)

	// rewind
	file.Seek(0, 0)

	matchedValues := make([][]string, len(regexRules)+1)
	count := 0

	// skip binary
	// TODO better method of detection
	if kind.MIME.Type == "video" || kind.MIME.Value == "application/x-bzip2" {
		return matchedValues, count
	} else if kind.MIME.Value == "application/pdf" {
		pdfReader, err := pdf.NewPdfReader(file)
		if err != nil {
			log.Fatal(err)
		}

		numPages, err := pdfReader.GetNumPages()
		if err != nil {
			log.Fatal(err)
		}

		content := ""

		for i := 0; i < numPages; i++ {
			pageNum := i + 1

			page, err := pdfReader.GetPage(pageNum)
			if err != nil {
				log.Fatal(err)
			}

			contentStreams, err := page.GetContentStreams()
			if err != nil {
				log.Fatal(err)
			}

			// If the value is an array, the effect shall be as if all of the streams in the array were concatenated,
			// in order, to form a single stream.
			pageContentStr := ""
			for _, cstream := range contentStreams {
				pageContentStr += cstream
			}

			cstreamParser := pdfcontent.NewContentStreamParser(pageContentStr)
			txt, err := cstreamParser.ExtractText()
			if err != nil {
				log.Fatal(err)
			}
			content += txt
		}

		scanner := bufio.NewScanner(strings.NewReader(content))
		return findScannerMatches(scanner)
	} else if kind.MIME.Value == "application/zip" {
		// TODO make zip work with S3
		reader, err := zip.OpenReader(filename)
		if err != nil {
			log.Fatal(err)
		}
		defer reader.Close()

		for _, file := range reader.File {
			if file.FileInfo().IsDir() {
				continue
			}

			fileReader, err := file.Open()
			if err != nil {
				log.Fatal(err)
			}
			defer fileReader.Close()

			scanner := bufio.NewScanner(fileReader)
			fileMatchedValues, fileCount := findScannerMatches(scanner)

			// TODO capture specific file in archive
			for i, _ := range matchedValues {
				matchedValues[i] = append(matchedValues[i], fileMatchedValues[i]...)
			}
			count += fileCount
		}

		return matchedValues, count
	} else if kind.MIME.Value == "application/gzip" {
		gz, err := gzip.NewReader(file)
		if err != nil {
			log.Fatal(err)
		}

		scanner := bufio.NewScanner(gz)
		return findScannerMatches(scanner)
	}

	scanner := bufio.NewScanner(file)
	return findScannerMatches(scanner)
}
