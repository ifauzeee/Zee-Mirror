package domain

type YTDLPSession struct {
	URL      string
	FileName string
	Zip      bool
	Password string
	Type     TaskType
}

type TorrentSession struct {
	URL            string
	FileName       string
	Zip            bool
	Unzip          bool
	Password       string
	SelectedFiles  []int
	ChatID         int64
	MessageID      int
	ReplyID        int
	UserID         int64
	Files          []TorrentFile
	Error          string
	IsFetching     bool
	StatusMessages []string
}

type TorrentFile struct {
	Index int    `json:"index"`
	Name  string `json:"name"`
	Size  int64  `json:"size"`
	Path  string `json:"path"`
}
