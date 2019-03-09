package internal

import (
	"archive/zip"
	"bufio"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/h2non/filetype"
)

type ReadSeekCloser interface {
	io.Reader
	io.Seeker
}

func findLocalFiles(urlStr string) []string {
	var files []string

	root := urlStr[7:]
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		abort(err)
	}

	return files
}

// TODO skip certain file types
func findFiles(urlStr string) []string {
	if strings.HasPrefix(urlStr, "file://") {
		return findLocalFiles(urlStr)
	}
	return findS3Files(urlStr)
}

func findScannerMatches(reader io.Reader) ([][]string, int) {
	matchedValues := make([][]string, len(regexRules)+1)
	nameIndex := len(regexRules)
	count := 0

	scanner := bufio.NewScanner(reader)
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
		file = downloadS3File(filename)
	} else {
		f, err := os.Open(filename)
		if err != nil {
			abort(err)
		}

		defer f.Close()

		file = f
	}

	return processFile(file, filename)
}

// TODO make zip work with S3
func processZip(filename string) ([][]string, int) {
	matchedValues := make([][]string, len(regexRules)+1)
	count := 0

	reader, err := zip.OpenReader(filename)
	if err != nil {
		abort(err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		fileReader, err := file.Open()
		if err != nil {
			abort(err)
		}
		defer fileReader.Close()

		// TODO recursively process files
		fileMatchedValues, fileCount := findScannerMatches(fileReader)

		// TODO capture specific file in archive
		for i, _ := range matchedValues {
			matchedValues[i] = append(matchedValues[i], fileMatchedValues[i]...)
		}
		count += fileCount
	}

	return matchedValues, count
}

func processGzip(file ReadSeekCloser) ([][]string, int) {
	gz, err := gzip.NewReader(file)
	if err != nil {
		abort(err)
	}

	return findScannerMatches(gz)
}

func processFile(file ReadSeekCloser, filename string) ([][]string, int) {
	// we only have to pass the file header = first 261 bytes
	head := make([]byte, 261)
	file.Read(head)
	kind, err := filetype.Match(head)
	if err != nil {
		abort(err)
	}
	// fmt.Println(kind.MIME.Value)

	// rewind
	file.Seek(0, 0)

	// skip binary
	// TODO better method of detection
	if kind.MIME.Type == "video" || kind.MIME.Value == "application/x-bzip2" {
		matchedValues := make([][]string, len(regexRules)+1)
		count := 0
		return matchedValues, count
	// } else if kind.MIME.Value == "application/pdf" {
	// 	return processPdf(file)
	} else if kind.MIME.Value == "application/zip" {
		return processZip(filename)
	} else if kind.MIME.Value == "application/gzip" {
		return processGzip(file)
	}

	return findScannerMatches(file)
}
