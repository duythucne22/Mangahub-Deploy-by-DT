package utils

import (
	"fmt"
	"strings"
)

// SafeError returns error message or default if nil
func SafeError(err error, defaultMsg string) string {
	if err != nil {
		return err.Error()
	}
	return defaultMsg
}

// CombineErrors combines multiple errors into one
func CombineErrors(errs ...error) error {
	var messages []string
	for _, err := range errs {
		if err != nil {
			messages = append(messages, err.Error())
		}
	}
	if len(messages) == 0 {
		return nil
	}
	return fmt.Errorf("multiple errors: %s", strings.Join(messages, "; "))
}