package internal

type FileAdapter interface {
	Init(url string) error
	FetchFiles() ([]string, error)
	FindFileMatches(file string) ([][]string, int, error)
}
