package internal

type FileAdapter interface {
	ObjectName() string
	Init(url string) error
	FetchFiles() ([]string, error)
	FindFileMatches(file string) ([][]string, int, error)
}
