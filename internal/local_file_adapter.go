package internal

import (
	"os"
	"path/filepath"
)

type LocalFileAdapter struct {
	url string
}

func (a *LocalFileAdapter) ObjectName() string {
	return "file"
}

func (a *LocalFileAdapter) Init(url string) error {
	a.url = url
	return nil
}

func (a LocalFileAdapter) FetchFiles() ([]string, error) {
	urlStr := a.url
	var files []string

	root := urlStr[7:]
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return files, err
	}

	return files, nil
}

// TODO read metadata for certain file types
func (a LocalFileAdapter) FindFileMatches(filename string, matchFinder *MatchFinder) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return processFile(f, matchFinder)
}
