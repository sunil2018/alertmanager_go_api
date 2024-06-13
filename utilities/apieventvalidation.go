package utilities

import (
	"time"
	"fmt"
	"errors"
)

type ApiAlertData map[string]interface{}

type CustomTime struct {
    time.Time
}

func (e ApiAlertData ) IsEmpty(keys ...string) (string, error) {
	fmt.Println(e)
	for _, key := range keys {
		if value, ok := e[key]; !ok || value == nil {
			return key, errors.New("key is missing or empty")
		} else if str, ok := value.(string); ok && str == "" {
			return key, errors.New("key is missing or empty")
		}
	}
	return "", nil
}