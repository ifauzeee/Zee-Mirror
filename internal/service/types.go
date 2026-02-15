package service

import "time"

type DriveFile struct {
	ID       string `json:"ID"`
	Name     string `json:"Name"`
	MimeType string `json:"MimeType"`
	Path     string `json:"Path"`
	ModTime  string `json:"ModTime"`
	Size     int64  `json:"Size"`
	IsDir    bool   `json:"IsDir"`
}

type Result struct {
	Title   string
	Size    string
	Seeders string
	Magnet  string
	Source  string
}

type Session struct {
	CreatedAt time.Time
	Query     string
	Provider  string
	Results   []Result
	Page      int
}
