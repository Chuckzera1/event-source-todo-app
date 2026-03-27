package domain

import "time"

type TaskType string

const (
	TaskTypeHomeTask  TaskType = "homework"
	TaskTypeStudyTask TaskType = "studytask"
	TaskTypeWorkTask  TaskType = "worktask"
	TaskTypeOther     TaskType = "other"
)

type Task struct {
	ID          string
	Title       string
	Description string
	Completed   bool
	Type        TaskType
	CreatedAt   time.Time
	UpdatedAt   time.Time
}