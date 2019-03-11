package internal

import (
	"os"
	"path/filepath"
)

type LocalFileAdapter struct {
	url string
}

func (a *LocalFileAdapter) Init(url string) {
	a.url = url
}

func (a LocalFileAdapter) FetchFiles() []string {
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
		abort(err)
	}

	return files
}

// TODO read metadata for certain file types
func (a LocalFileAdapter) FindFileMatches(filename string) ([][]string, int) {
	f, err := os.Open(filename)
	if err != nil {
		abort(err)
	}
	defer f.Close()

	return processFile(f)
}
