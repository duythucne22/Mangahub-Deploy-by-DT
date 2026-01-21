package utils

import (
	"fmt"
	"sync"
	"time"
)

// GenerateID creates schema-aligned IDs matching SPEC.md requirements
// Format: prefix-timestamp-counter (e.g., "user-1705612800000-001")
func GenerateID(prefix string) string {
	timestamp := time.Now().UnixMilli()
	counter := atomicCounter()
	return fmt.Sprintf("%s-%d-%03d", prefix, timestamp, counter)
}

// GenerateMangaID creates manga-specific ID
func GenerateMangaID() string {
	return GenerateID("manga")
}

// GenerateUserID creates user-specific ID
func GenerateUserID() string {
	return GenerateID("user")
}

// GenerateActivityID creates activity-specific ID
func GenerateActivityID() string {
	return GenerateID("act")
}

// atomicCounter provides thread-safe sequential counters
var (
	counter int64
	mu      sync.Mutex
)

func atomicCounter() int {
	mu.Lock()
	defer mu.Unlock()
	counter++
	return int(counter)
}