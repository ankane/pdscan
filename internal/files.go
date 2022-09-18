package internal

import (
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"io"

	"github.com/h2non/filetype"
)

func findScannerMatches(reader io.Reader, matchFinder *MatchFinder) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		v := scanner.Text()
		// TODO pass archive file and line number in file
		matchFinder.Scan(v, matchFinder.Count)
		matchFinder.Count += 1
	}
	return nil
}

func processZip(file io.Reader, matchFinder *MatchFinder) error {
	// TODO make more efficient
	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	bytesFile := bytes.NewReader(data)
	size := int64(bytesFile.Size())

	reader, err := zip.NewReader(bytesFile, size)
	if err != nil {
		return err
	}

	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		fileReader, err := file.Open()
		if err != nil {
			return err
		}
		defer fileReader.Close()

		// TODO capture specific file in archive
		err = processFile(fileReader, matchFinder)
		if err != nil {
			return err
		}
	}

	return nil
}

func processGzip(file io.Reader, matchFinder *MatchFinder) error {
	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}

	return findScannerMatches(gz, matchFinder)
}

func processFile(file io.Reader, matchFinder *MatchFinder) error {
	reader := bufio.NewReader(file)

	// we only have to pass the file header = first 261 bytes
	head, err := reader.Peek(261)
	if err != nil && err != io.EOF {
		return err
	}

	kind, err := filetype.Match(head)
	if err == filetype.ErrEmptyBuffer {
		return nil
	} else if err != nil {
		return err
	}
	// fmt.Println(kind.MIME.Value)

	// skip binary
	// TODO better method of detection
	if kind.MIME.Type == "video" || kind.MIME.Value == "application/x-bzip2" {
		return nil
		// } else if kind.MIME.Value == "application/pdf" {
		// 	return processPdf(file)
	} else if kind.MIME.Value == "application/zip" {
		return processZip(reader, matchFinder)
	} else if kind.MIME.Value == "application/gzip" {
		return processGzip(reader, matchFinder)
	}

	return findScannerMatches(reader, matchFinder)
}
