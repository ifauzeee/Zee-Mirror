package domain

type YTDLPSession struct {
	URL      string
	FileName string
	Zip      bool
	Password string
	Type     TaskType
}
