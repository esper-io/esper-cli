package runtime

import (
	"errors"
	"fmt"
)

type Category string

const (
	CategoryAPI     Category = "api"
	CategoryUsage   Category = "usage"
	CategoryAuth    Category = "auth"
	CategoryNetwork Category = "network"
)

var (
	ErrAPI     = errors.New("API error")
	ErrUsage   = errors.New("usage error")
	ErrAuth    = errors.New("authentication or configuration error")
	ErrNetwork = errors.New("network error")
)

type CategorizedError struct {
	Category Category
	Err      error
}

func (err *CategorizedError) Error() string { return err.Err.Error() }
func (err *CategorizedError) Unwrap() error { return err.Err }

func NewError(category Category, err error) error {
	if err == nil {
		return nil
	}
	return &CategorizedError{Category: category, Err: err}
}

func ErrorCategory(err error) Category {
	var categorized *CategorizedError
	if errors.As(err, &categorized) {
		return categorized.Category
	}
	switch {
	case errors.Is(err, ErrUsage):
		return CategoryUsage
	case errors.Is(err, ErrAuth):
		return CategoryAuth
	case errors.Is(err, ErrNetwork):
		return CategoryNetwork
	default:
		return CategoryAPI
	}
}

func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	switch ErrorCategory(err) {
	case CategoryUsage:
		return 2
	case CategoryAuth:
		return 3
	case CategoryNetwork:
		return 4
	default:
		return 1
	}
}

func FormatError(err error) string {
	return fmt.Sprintf("error: %s: %s", ErrorCategory(err), err)
}
