package presenters

import (
	"regexp"
	"strings"
)

type SnakeCasePresenter struct {
}

func NewSnakeCasePresenter() SnakeCasePresenter {
	return SnakeCasePresenter{}
}

func (p SnakeCasePresenter) Present(str string) string {
	regex := regexp.MustCompile("([^A-Z])([A-Z])")

	str = regex.ReplaceAllString(str, "${1}_${2}")
	str = strings.ToLower(str)

	return str
}
