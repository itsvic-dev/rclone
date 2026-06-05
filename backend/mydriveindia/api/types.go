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

type CreateFolderRequest struct {
	Path string `json:"path"`
}

type CreateFileResponse struct {
	File FileInfo `json:"file"`
}

type DownloadTokenResponse struct {
	DownloadToken string `json:"download_token"`
}

type MPIUser struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	IsAdmin   bool   `json:"is_admin"`
}

type User struct {
	Username       string  `json:"username"`
	SpaceAvailable int64   `json:"space_available"`
	SpaceUsed      int64   `json:"space_used"`
	MPIUser        MPIUser `json:"mpi"`
}

type UserInfoResponse struct {
	User User `json:"user"`
}
