package internal

type FileAdapter interface {
	ObjectName() string
	Init(url string) error
	FetchFiles() ([]string, error)
	FindFileMatches(file string, matchFinder *MatchFinder) error
}
