package repository

import (
    "fmt"
    "time"
)

func generateUUID(prefix string) string {
    return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}