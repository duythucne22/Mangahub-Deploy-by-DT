package tcp

import "time"

type ProgressUpdate struct {
	UserID    string `json:"user_id"`
	MangaID   string `json:"manga_id"`
	Chapter   int    `json:"chapter"`
	Timestamp int64  `json:"timestamp"`
}

func NewProgressUpdate(userID, mangaID string, chapter int) ProgressUpdate {
	return ProgressUpdate{
		UserID:    userID,
		MangaID:   mangaID,
		Chapter:   chapter,
		Timestamp: time.Now().Unix(),
	}
}
