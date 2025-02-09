package task

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// TaskStore manages a collection of tasks.
type TaskStore struct {
	Tasks  map[int]Task `json:"tasks"` // Map of tasks (key: task ID)
	NextID int          `json:"next_id"` // Next ID to assign to a new task
	mu     sync.Mutex   // Mutex to ensure thread safety
}

// NewTaskStore initializes a new TaskStore.
func NewTaskStore() *TaskStore {
	return &TaskStore{
		Tasks:  make(map[int]Task),
		NextID: 1,
	}
}

// AddTask adds a new task to the store.
func (ts *TaskStore) AddTask(name string, priority string, dueDate time.Time) (Task, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	task := Task{
		ID:        ts.NextID,
		Name:      name,
		Priority:  priority,
		DueDate:   dueDate,
		CreatedAt: time.Now(),
	}

	if err := task.Validate(); err != nil {
		return Task{}, fmt.Errorf("invalid task: %w", err)
	}

	ts.Tasks[ts.NextID] = task
	ts.NextID++
	return task, nil
}

// SaveToFile saves the tasks to a JSON file.
func (ts *TaskStore) SaveToFile(filename string) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Pretty-print JSON
	if err := encoder.Encode(ts); err != nil {
		return fmt.Errorf("failed to encode tasks: %w", err)
	}

	return nil
}

// LoadFromFile loads tasks from a JSON file.
func (ts *TaskStore) LoadFromFile(filename string) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet, no tasks to load
		}
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(ts); err != nil {
		return fmt.Errorf("failed to decode tasks: %w", err)
	}

	return nil
}