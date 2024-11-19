package data

import "strings"

type Errors []error

func (list Errors) Error() string {
	errs := make([]string, 0, len(list))
	for _, e := range list {
		errs = append(errs, e.Error())
	}
	return strings.Join(errs, ",")
}

func (list Errors) HasAny() bool {
	return len(list) > 0
}

func (list Errors) FirstError() error {
	if len(list) > 0 {
		return list[0]
	}
	return nil
}
