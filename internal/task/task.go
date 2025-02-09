package task

import (
	"errors"
	"time"
)

// Task represents a to-do task.
type Task struct {
	ID          int       `json:"id"`           // Unique identifier for the task
	Name        string    `json:"name"`         // Name/description of the task
	Completed   bool      `json:"completed"`    // Whether the task is completed
	DueDate     time.Time `json:"due_date"`     // Optional due date for the task
	Priority    string    `json:"priority"`     // Priority level (e.g., low, medium, high)
	CreatedAt   time.Time `json:"created_at"`   // When the task was created
	CompletedAt time.Time `json:"completed_at"` // When the task was completed
}

// Validate checks if a task is valid.
func (t *Task) Validate() error {
	if t.Name == "" {
		return errors.New("task name cannot be empty")
	}
	if t.Priority != "" && t.Priority != "low" && t.Priority != "medium" && t.Priority != "high" {
		return errors.New("priority must be low, medium, or high")
	}
	return nil
}

// MarkComplete marks the task as completed.
func (t *Task) MarkComplete() {
	t.Completed = true
	t.CompletedAt = time.Now()
}

// MarkIncomplete marks the task as incomplete.
func (t *Task) MarkIncomplete() {
	t.Completed = false
	t.CompletedAt = time.Time{} // Reset completed time
}