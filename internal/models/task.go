package models

import (
	"errors"
	"time"
)

// Priority represents the importance level of a task
type Priority int

const (
	Low Priority = iota
	Medium
	High
	Urgent
)

// String returns the string representation of Priority
func (p Priority) String() string {
	switch p {
	case Low:
		return "Low"
	case Medium:
		return "Medium"
	case High:
		return "High"
	case Urgent:
		return "Urgent"
	default:
		return "Unknown"
	}
}

// TaskStatus represents the current status of a task
type TaskStatus int

const (
	NotStarted TaskStatus = iota
	InProgress
	Completed
	Archived
)

// String returns the string representation of TaskStatus
func (s TaskStatus) String() string {
	switch s {
	case NotStarted:
		return "Not Started"
	case InProgress:
		return "In Progress"
	case Completed:
		return "Completed"
	case Archived:
		return "Archived"
	default:
		return "Unknown"
	}
}

// SubTask represents a smaller component of a main task
type SubTask struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

// Task represents a todo item with enhanced features for students
type Task struct {
	ID           int         `json:"id"`
	Name         string      `json:"name"`
	Description  string      `json:"description,omitempty"`
	Status       TaskStatus  `json:"status"`
	Priority     Priority    `json:"priority"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
	DueDate      time.Time   `json:"due_date,omitempty"`
	CompletedAt  time.Time   `json:"completed_at,omitempty"`
	EstimatedMin int         `json:"estimated_minutes,omitempty"`
	ActualMin    int         `json:"actual_minutes,omitempty"`
	Tags         []string    `json:"tags,omitempty"`
	Category     string      `json:"category,omitempty"`
	SubTasks     []SubTask   `json:"subtasks,omitempty"`
	Notes        string      `json:"notes,omitempty"`
	References   []string    `json:"references,omitempty"`
	Progress     int         `json:"progress"`          // 0-100%
	Reminder     *time.Time  `json:"reminder,omitempty"`
	SharedWith   []string    `json:"shared_with,omitempty"`
}

// NewTask creates a new task with default values
func NewTask(name string) *Task {
	return &Task{
		Name:      name,
		Status:    NotStarted,
		Priority:  Medium,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Progress:  0,
		Tags:      make([]string, 0),
		SubTasks:  make([]SubTask, 0),
		References: make([]string, 0),
		SharedWith: make([]string, 0),
	}
}

// Validate checks if the task has valid data
func (t *Task) Validate() error {
	if t.Name == "" {
		return errors.New("task name cannot be empty")
	}

	if t.Progress < 0 || t.Progress > 100 {
		return errors.New("progress must be between 0 and 100")
	}

	if !t.DueDate.IsZero() && t.DueDate.Before(time.Now()) {
		return errors.New("due date cannot be in the past")
	}

	return nil
}

// Complete marks the task as completed
func (t *Task) Complete() {
	t.Status = Completed
	t.Progress = 100
	t.CompletedAt = time.Now()
	t.UpdatedAt = time.Now()
}

// UpdateProgress updates the progress of the task
func (t *Task) UpdateProgress(progress int) error {
	if progress < 0 || progress > 100 {
		return errors.New("progress must be between 0 and 100")
	}
	t.Progress = progress
	t.UpdatedAt = time.Now()
	
	if progress == 100 && t.Status != Completed {
		t.Complete()
	} else if progress < 100 && t.Status == NotStarted {
		t.Status = InProgress
	}
	
	return nil
}

// AddSubTask adds a new subtask to the task
func (t *Task) AddSubTask(name string) {
	subTask := SubTask{
		ID:        len(t.SubTasks) + 1,
		Name:      name,
		CreatedAt: time.Now(),
		Completed: false,
	}
	t.SubTasks = append(t.SubTasks, subTask)
	t.UpdatedAt = time.Now()
}

// CompleteSubTask marks a subtask as completed
func (t *Task) CompleteSubTask(id int) error {
	for i, st := range t.SubTasks {
		if st.ID == id {
			t.SubTasks[i].Completed = true
			t.SubTasks[i].CompletedAt = time.Now()
			t.UpdatedAt = time.Now()
			
			// Update overall progress based on completed subtasks
			completedCount := 0
			for _, subTask := range t.SubTasks {
				if subTask.Completed {
					completedCount++
				}
			}
			t.Progress = (completedCount * 100) / len(t.SubTasks)
			return nil
		}
	}
	return errors.New("subtask not found")
}

// AddTag adds a new tag to the task
func (t *Task) AddTag(tag string) {
	// Check if tag already exists
	for _, existingTag := range t.Tags {
		if existingTag == tag {
			return
		}
	}
	t.Tags = append(t.Tags, tag)
	t.UpdatedAt = time.Now()
}

// RemoveTag removes a tag from the task
func (t *Task) RemoveTag(tag string) {
	for i, existingTag := range t.Tags {
		if existingTag == tag {
			t.Tags = append(t.Tags[:i], t.Tags[i+1:]...)
			t.UpdatedAt = time.Now()
			return
		}
	}
}

// IsOverdue checks if the task is past its due date
func (t *Task) IsOverdue() bool {
	return !t.DueDate.IsZero() && time.Now().After(t.DueDate)
}

// TimeUntilDue returns the duration until the task is due
func (t *Task) TimeUntilDue() (time.Duration, error) {
	if t.DueDate.IsZero() {
		return 0, errors.New("no due date set")
	}
	return t.DueDate.Sub(time.Now()), nil
}

// SetReminder sets a reminder for the task
func (t *Task) SetReminder(reminderTime time.Time) error {
	if reminderTime.Before(time.Now()) {
		return errors.New("reminder time cannot be in the past")
	}
	t.Reminder = &reminderTime
	t.UpdatedAt = time.Now()
	return nil
}

// ShareWith shares the task with other users
func (t *Task) ShareWith(users []string) {
	for _, user := range users {
		// Check if user is already in SharedWith
		alreadyShared := false
		for _, existingUser := range t.SharedWith {
			if existingUser == user {
				alreadyShared = true
				break
			}
		}
		if !alreadyShared {
			t.SharedWith = append(t.SharedWith, user)
		}
	}
	t.UpdatedAt = time.Now()
}

// UnshareWith removes users from the shared list
func (t *Task) UnshareWith(users []string) {
	for _, user := range users {
		for i, existingUser := range t.SharedWith {
			if existingUser == user {
				t.SharedWith = append(t.SharedWith[:i], t.SharedWith[i+1:]...)
				break
			}
		}
	}
	t.UpdatedAt = time.Now()
}