package domain

type ScheduledTask struct {
	ID          string
	TaskType    string
	URL         string
	FileName    string
	ChatID      int64
	UserID      int64
	Zip         bool
	Unzip       bool
	Password    string
	Quality     string
	ScheduledAt string
	Status      string
	TaskID      string
	CreatedAt   string
}

type ScheduledTaskFilter struct {
	Page   int
	Limit  int
	Status string
}
