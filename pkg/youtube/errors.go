package youtube

import (
	"errors"
	"fmt"
)

var (
	ErrEmptyQuery = errors.New("empty query")
	ErrNoResults  = errors.New("no results found")
)

type APIError struct {
	Status int
	Body   string
}

func (e APIError) Error() string {
	return fmt.Sprintf("youtube music api error: status=%d body=%q", e.Status, e.Body)
}
