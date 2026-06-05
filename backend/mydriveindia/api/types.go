package api

import (
	"fmt"
	"time"
)

type Error struct {
	Message string `json:"message"`
}

func (e Error) Error() string {
	return fmt.Sprintf("Error %q", e.Message)
}

type FileInfo struct {
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

type FilesResponse struct {
	Files []FileInfo `json:"files"`
}
