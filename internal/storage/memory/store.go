package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/tiwariParth/go-todo-cli/internal/models"
	"github.com/tiwariParth/go-todo-cli/internal/storage"
)

// MemoryStore implements the storage.Storage interface using in-memory storage
type MemoryStore struct {
	tasks    map[int]models.Task
	maxID    int
	mu       sync.RWMutex
	isActive bool
}

// NewMemoryStore creates a new instance of MemoryStore
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		tasks:    make(map[int]models.Task),
		maxID:    0,
		isActive: true,
	}
}

// Connect initializes the memory store
func (m *MemoryStore) Connect() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isActive {
		return fmt.Errorf("store is already connected")
	}
	m.isActive = true
	return nil
}

// Close cleans up resources
func (m *MemoryStore) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isActive {
		return fmt.Errorf("store is already closed")
	}
	m.isActive = false
	return nil
}

// Ping checks if the store is active
func (m *MemoryStore) Ping(ctx context.Context) error {
	if !m.isActive {
		return storage.ErrStorageConnection
	}
	return nil
}

// CreateTask adds a new task to the store
func (m *MemoryStore) CreateTask(ctx context.Context, task *models.Task) error {
	if err := m.checkActive(); err != nil {
		return err
	}

	if err := task.Validate(); err != nil {
		return fmt.Errorf("%w: %v", storage.ErrTaskValidation, err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.maxID++
	task.ID = m.maxID
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()

	m.tasks[task.ID] = *task
	return nil
}

// GetTask retrieves a task by ID
func (m *MemoryStore) GetTask(ctx context.Context, id int) (*models.Task, error) {
	if err := m.checkActive(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	task, exists := m.tasks[id]
	if !exists {
		return nil, storage.ErrTaskNotFound
	}

	return &task, nil
}

// UpdateTask updates an existing task
func (m *MemoryStore) UpdateTask(ctx context.Context, task *models.Task) error {
	if err := m.checkActive(); err != nil {
		return err
	}

	if err := task.Validate(); err != nil {
		return fmt.Errorf("%w: %v", storage.ErrTaskValidation, err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tasks[task.ID]; !exists {
		return storage.ErrTaskNotFound
	}

	task.UpdatedAt = time.Now()
	m.tasks[task.ID] = *task
	return nil
}

// DeleteTask removes a task by ID
func (m *MemoryStore) DeleteTask(ctx context.Context, id int) error {
	if err := m.checkActive(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tasks[id]; !exists {
		return storage.ErrTaskNotFound
	}

	delete(m.tasks, id)
	return nil
}

// ListTasks returns tasks based on filter, sort, and pagination options
func (m *MemoryStore) ListTasks(ctx context.Context, filter *storage.Filter, sort *storage.SortOption, page *storage.Page) ([]models.Task, error) {
	if err := m.checkActive(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Convert map to slice for filtering and sorting
	var tasks []models.Task
	for _, task := range m.tasks {
		if m.matchesFilter(task, filter) {
			tasks = append(tasks, task)
		}
	}

	// Sort tasks
	if sort != nil {
		m.sortTasks(tasks, sort)
	}

	// Apply pagination
	if page != nil {
		return m.paginateTasks(tasks, page), nil
	}

	return tasks, nil
}

// SearchTasks performs a simple search across task fields
func (m *MemoryStore) SearchTasks(ctx context.Context, query string) ([]models.Task, error) {
	if err := m.checkActive(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	query = strings.ToLower(query)
	var results []models.Task

	for _, task := range m.tasks {
		if strings.Contains(strings.ToLower(task.Name), query) ||
			strings.Contains(strings.ToLower(task.Description), query) ||
			strings.Contains(strings.ToLower(task.Category), query) {
			results = append(results, task)
		}
	}

	return results, nil
}

// GetTasksByCategory returns tasks for a specific category
func (m *MemoryStore) GetTasksByCategory(ctx context.Context, category string) ([]models.Task, error) {
	if err := m.checkActive(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var tasks []models.Task
	for _, task := range m.tasks {
		if task.Category == category {
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

// GetTaskSummary returns statistics about tasks
func (m *MemoryStore) GetTaskSummary(ctx context.Context) (*storage.TaskSummary, error) {
	if err := m.checkActive(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	summary := &storage.TaskSummary{
		TasksByCategory: make(map[string]int),
		TasksByPriority: make(map[models.Priority]int),
	}

	now := time.Now()
	for _, task := range m.tasks {
		summary.TotalTasks++

		if task.Status == models.Completed {
			summary.CompletedTasks++
		} else {
			summary.PendingTasks++
		}

		if !task.DueDate.IsZero() && task.DueDate.Before(now) && task.Status != models.Completed {
			summary.OverdueTasks++
		}

		if task.Category != "" {
			summary.TasksByCategory[task.Category]++
		}
		summary.TasksByPriority[task.Priority]++

		// Collect upcoming deadlines (next 7 days)
		if !task.DueDate.IsZero() && task.DueDate.After(now) && task.DueDate.Before(now.AddDate(0, 0, 7)) {
			summary.UpcomingDeadlines = append(summary.UpcomingDeadlines, task)
		}
	}

	return summary, nil
}

// Helper functions

func (m *MemoryStore) checkActive() error {
	if !m.isActive {
		return storage.ErrStorageConnection
	}
	return nil
}

func (m *MemoryStore) matchesFilter(task models.Task, filter *storage.Filter) bool {
	if filter == nil {
		return true
	}

	if filter.Status != nil && task.Status != *filter.Status {
		return false
	}

	if filter.Priority != nil && task.Priority != *filter.Priority {
		return false
	}

	if filter.Category != "" && task.Category != filter.Category {
		return false
	}

	if len(filter.Tags) > 0 {
		hasTag := false
		for _, filterTag := range filter.Tags {
			for _, taskTag := range task.Tags {
				if filterTag == taskTag {
					hasTag = true
					break
				}
			}
			if hasTag {
				break
			}
		}
		if !hasTag {
			return false
		}
	}

	if filter.DueBefore != nil && !task.DueDate.Before(*filter.DueBefore) {
		return false
	}

	if filter.DueAfter != nil && !task.DueDate.After(*filter.DueAfter) {
		return false
	}

	if filter.IsOverdue && !task.IsOverdue() {
		return false
	}

	return true
}

func (m *MemoryStore) sortTasks(tasks []models.Task, sort *storage.SortOption) {
	if sort == nil {
		return
	}

	sort.Field = strings.ToLower(sort.Field)
	sort.Ascending = true

	sortFunc := func(i, j int) bool {
		var result bool
		switch sort.Field {
		case "due_date":
			result = tasks[i].DueDate.Before(tasks[j].DueDate)
		case "priority":
			result = tasks[i].Priority < tasks[j].Priority
		case "created_at":
			result = tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
		case "name":
			result = tasks[i].Name < tasks[j].Name
		default:
			result = tasks[i].ID < tasks[j].ID
		}
		if !sort.Ascending {
			return !result
		}
		return result
	}

	sort.Sort(taskSorter{tasks, sortFunc})
}

func (m *MemoryStore) paginateTasks(tasks []models.Task, page *storage.Page) []models.Task {
	if page == nil || page.Limit <= 0 {
		return tasks
	}

	start := page.Offset
	if start >= len(tasks) {
		return []models.Task{}
	}

	end := start + page.Limit
	if end > len(tasks) {
		end = len(tasks)
	}

	return tasks[start:end]
}

// taskSorter implements sort.Interface for []models.Task
type taskSorter struct {
	tasks []models.Task
	less  func(i, j int) bool
}

func (s taskSorter) Len() int           { return len(s.tasks) }
func (s taskSorter) Less(i, j int) bool { return s.less(i, j) }
func (s taskSorter) Swap(i, j int)      { s.tasks[i], s.tasks[j] = s.tasks[j], s.tasks[i] }