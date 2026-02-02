package domain

type YTDLPSession struct {
	URL      string
	FileName string
	Zip      bool
	Password string
	Type     TaskType
}

// TorrentSession stores torrent download session information
type TorrentSession struct {
	URL           string // Magnet link or torrent file URL
	FileName      string // Custom filename if provided
	Zip           bool   // Whether to zip the result
	Unzip         bool   // Whether to unzip archives
	Password      string // Archive password
	SelectedFiles []int  // Selected file indices (empty = all files)
	ChatID        int64  // Chat ID for the session
	MessageID     int    // Message ID for updates
	ReplyID       int    // Reply message ID
	UserID        int64  // User ID who initiated
}

// TorrentFile represents a file within a torrent
type TorrentFile struct {
	Index int    `json:"index"`
	Name  string `json:"name"`
	Size  int64  `json:"size"`
	Path  string `json:"path"`
}
