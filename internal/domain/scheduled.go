package domain

type ScheduledTask struct {
	ID          string
	TaskType    string
	URL         string
	FileName    string
	Password    string
	Quality     string
	ScheduledAt string
	Status      string
	TaskID      string
	CreatedAt   string
	ChatID      int64
	UserID      int64
	Zip         bool
	Unzip       bool
}

type ScheduledTaskFilter struct {
	Status string
	Page   int
	Limit  int
}
