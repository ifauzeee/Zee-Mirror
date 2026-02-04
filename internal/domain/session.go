package domain

type YTDLPSession struct {
	URL      string
	FileName string
	Password string
	Type     TaskType
	Zip      bool
}

type TorrentSession struct {
	URL            string
	FileName       string
	Password       string
	Error          string
	Files          []TorrentFile
	SelectedFiles  []int
	StatusMessages []string
	ChatID         int64
	UserID         int64
	MessageID      int
	ReplyID        int
	Zip            bool
	Unzip          bool
	IsFetching     bool
}

type TorrentFile struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Size  int64  `json:"size"`
	Index int    `json:"index"`
}
