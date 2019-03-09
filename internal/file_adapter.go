package internal

type FileAdapter interface {
  Init(url string)
  FetchFiles() (files []string)
  FindFileMatches(file string) ([][]string, int)
}
