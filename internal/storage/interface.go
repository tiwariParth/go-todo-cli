package storage

import (
	"context"
	"errors"
	"time"

	"github.com/tiwariParth/go-todo-cli/internal/models"
	"github.com/tiwariParth/go-todo-cli/internal/storage/memory"
)

// Common errors that can be returned by any storage implementation
var (
	ErrTaskNotFound      = errors.New("task not found")
	ErrInvalidTask       = errors.New("invalid task data")
	ErrDuplicateTask     = errors.New("task with this ID already exists")
	ErrStorageConnection = errors.New("storage connection error")
	ErrTaskValidation    = errors.New("task validation failed")
)

// Filter represents the filtering options for task queries
type Filter struct {
	Status     *models.TaskStatus
	Priority   *models.Priority
	Category   string
	Tags       []string
	DueBefore  *time.Time
	DueAfter   *time.Time
	IsOverdue  bool
	SearchTerm string
}

// SortOption defines how tasks should be sorted
type SortOption struct {
	Field     string // "due_date", "priority", "created_at", "name"
	Ascending bool
}

func (s *SortOption) Sort(sorter memory.taskSorter) {
	panic("unimplemented")
}

// Page represents pagination parameters
type Page struct {
	Offset int
	Limit  int
}

// TaskSummary represents summarized task statistics
type TaskSummary struct {
	TotalTasks        int
	CompletedTasks    int
	PendingTasks      int
	OverdueTasks      int
	UpcomingDeadlines []models.Task
	TasksByCategory   map[string]int
	TasksByPriority   map[models.Priority]int
}

// Storage defines the interface for task storage operations
type Storage interface {
	// Core CRUD Operations
	CreateTask(ctx context.Context, task *models.Task) error
	GetTask(ctx context.Context, id int) (*models.Task, error)
	UpdateTask(ctx context.Context, task *models.Task) error
	DeleteTask(ctx context.Context, id int) error

	// Query Operations
	ListTasks(ctx context.Context, filter *Filter, sort *SortOption, page *Page) ([]models.Task, error)
	SearchTasks(ctx context.Context, query string) ([]models.Task, error)

	// Batch Operations
	CreateTasks(ctx context.Context, tasks []models.Task) error
	DeleteTasks(ctx context.Context, ids []int) error

	// Category Operations
	GetTasksByCategory(ctx context.Context, category string) ([]models.Task, error)
	GetCategories(ctx context.Context) ([]string, error)

	// Tag Operations
	GetTasksByTag(ctx context.Context, tag string) ([]models.Task, error)
	GetTags(ctx context.Context) ([]string, error)

	// Status Operations
	GetTasksByStatus(ctx context.Context, status models.TaskStatus) ([]models.Task, error)
	MarkTaskComplete(ctx context.Context, id int) error
	MarkTaskIncomplete(ctx context.Context, id int) error

	// Due Date Operations
	GetOverdueTasks(ctx context.Context) ([]models.Task, error)
	GetUpcomingTasks(ctx context.Context, days int) ([]models.Task, error)

	// Subtask Operations
	AddSubTask(ctx context.Context, taskID int, subtask models.SubTask) error
	UpdateSubTask(ctx context.Context, taskID int, subtask models.SubTask) error
	DeleteSubTask(ctx context.Context, taskID, subtaskID int) error

	// Statistics and Analytics
	GetTaskSummary(ctx context.Context) (*TaskSummary, error)
	GetProductivityStats(ctx context.Context, startDate, endDate time.Time) (map[string]interface{}, error)

	// Collaboration
	GetSharedTasks(ctx context.Context, userID string) ([]models.Task, error)
	ShareTask(ctx context.Context, taskID int, userIDs []string) error
	UnshareTask(ctx context.Context, taskID int, userIDs []string) error

	// Data Management
	Export(ctx context.Context, format string) ([]byte, error)
	Import(ctx context.Context, data []byte, format string) error
	Backup(ctx context.Context) error
	Restore(ctx context.Context, backupID string) error

	// Maintenance
	Clean(ctx context.Context, olderThan time.Time) error
	Vacuum(ctx context.Context) error

	// Connection Management
	Connect() error
	Close() error
	Ping(ctx context.Context) error
}

// TaskFilter helps build Filter objects with a fluent interface
type TaskFilter struct {
	filter Filter
}

// NewTaskFilter creates a new TaskFilter
func NewTaskFilter() *TaskFilter {
	return &TaskFilter{}
}

// WithStatus adds status filter
func (tf *TaskFilter) WithStatus(status models.TaskStatus) *TaskFilter {
	tf.filter.Status = &status
	return tf
}

// WithPriority adds priority filter
func (tf *TaskFilter) WithPriority(priority models.Priority) *TaskFilter {
	tf.filter.Priority = &priority
	return tf
}

// WithCategory adds category filter
func (tf *TaskFilter) WithCategory(category string) *TaskFilter {
	tf.filter.Category = category
	return tf
}

// WithTags adds tags filter
func (tf *TaskFilter) WithTags(tags ...string) *TaskFilter {
	tf.filter.Tags = tags
	return tf
}

// WithDueRange adds due date range filter
func (tf *TaskFilter) WithDueRange(after, before time.Time) *TaskFilter {
	tf.filter.DueAfter = &after
	tf.filter.DueBefore = &before
	return tf
}

// WithOverdue adds overdue filter
func (tf *TaskFilter) WithOverdue(overdue bool) *TaskFilter {
	tf.filter.IsOverdue = overdue
	return tf
}

// WithSearchTerm adds search term filter
func (tf *TaskFilter) WithSearchTerm(term string) *TaskFilter {
	tf.filter.SearchTerm = term
	return tf
}

// Build creates the final Filter
func (tf *TaskFilter) Build() *Filter {
	return &tf.filter
}
