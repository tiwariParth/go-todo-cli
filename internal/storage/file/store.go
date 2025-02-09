package file

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tiwariParth/go-todo-cli/internal/models"
	"github.com/tiwariParth/go-todo-cli/internal/storage"
)

// FileStore implements the storage.Storage interface using file-based storage
type FileStore struct {
	filePath     string
	tasks        map[int]models.Task
	maxID        int
	mu           sync.RWMutex
	isActive     bool
	lastSave     time.Time
	autoSaveTime time.Duration
}

// FileMetadata stores metadata about the task storage
type FileMetadata struct {
	Version     string    `json:"version"`
	LastUpdated time.Time `json:"last_updated"`
	TaskCount   int       `json:"task_count"`
	MaxID       int       `json:"max_id"`
}

// FileData represents the structure of the stored JSON file
type FileData struct {
	Metadata FileMetadata      `json:"metadata"`
	Tasks    []models.Task     `json:"tasks"`
	Backup   map[string][]byte `json:"backup,omitempty"`
}

// NewFileStore creates a new instance of FileStore
func NewFileStore(filePath string) (*FileStore, error) {
	if filePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		filePath = filepath.Join(homeDir, ".todo-cli", "tasks.json")
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	return &FileStore{
		filePath:     filePath,
		tasks:        make(map[int]models.Task),
		autoSaveTime: 5 * time.Minute,
	}, nil
}

// Connect loads the tasks from the file
func (f *FileStore) Connect() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.isActive {
		return fmt.Errorf("store is already connected")
	}

	// Create file if it doesn't exist
	if _, err := os.Stat(f.filePath); os.IsNotExist(err) {
		initialData := FileData{
			Metadata: FileMetadata{
				Version:     "1.0",
				LastUpdated: time.Now(),
				TaskCount:   0,
				MaxID:       0,
			},
			Tasks: []models.Task{},
		}
		if err := f.saveToFile(initialData); err != nil {
			return fmt.Errorf("failed to initialize file: %w", err)
		}
	}

	// Load existing data
	if err := f.loadFromFile(); err != nil {
		return fmt.Errorf("failed to load data: %w", err)
	}

	f.isActive = true
	go f.autoSaveRoutine()
	return nil
}

// Close saves the current state and closes the store
func (f *FileStore) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.isActive {
		return fmt.Errorf("store is already closed")
	}

	if err := f.save(); err != nil {
		return fmt.Errorf("failed to save on close: %w", err)
	}

	f.isActive = false
	return nil
}

// CreateTask adds a new task and persists it to file
func (f *FileStore) CreateTask(ctx context.Context, task *models.Task) error {
	if err := f.checkActive(); err != nil {
		return err
	}

	if err := task.Validate(); err != nil {
		return fmt.Errorf("%w: %v", storage.ErrTaskValidation, err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.maxID++
	task.ID = f.maxID
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()

	f.tasks[task.ID] = *task

	return f.saveIfNeeded()
}

// GetTask retrieves a task by ID
func (f *FileStore) GetTask(ctx context.Context, id int) (*models.Task, error) {
	if err := f.checkActive(); err != nil {
		return nil, err
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	task, exists := f.tasks[id]
	if !exists {
		return nil, storage.ErrTaskNotFound
	}

	return &task, nil
}

// UpdateTask updates an existing task
func (f *FileStore) UpdateTask(ctx context.Context, task *models.Task) error {
	if err := f.checkActive(); err != nil {
		return err
	}

	if err := task.Validate(); err != nil {
		return fmt.Errorf("%w: %v", storage.ErrTaskValidation, err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.tasks[task.ID]; !exists {
		return storage.ErrTaskNotFound
	}

	task.UpdatedAt = time.Now()
	f.tasks[task.ID] = *task

	return f.saveIfNeeded()
}

// DeleteTask removes a task
func (f *FileStore) DeleteTask(ctx context.Context, id int) error {
	if err := f.checkActive(); err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.tasks[id]; !exists {
		return storage.ErrTaskNotFound
	}

	delete(f.tasks, id)
	return f.saveIfNeeded()
}

// Backup creates a backup of the current state
func (f *FileStore) Backup(ctx context.Context) error {
	if err := f.checkActive(); err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	backupPath := f.filePath + ".backup." + time.Now().Format("20060102150405")
	data := f.prepareFileData()
	
	backupData, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal backup data: %w", err)
	}

	if err := os.WriteFile(backupPath, backupData, 0644); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	return nil
}

// Restore restores from a backup
func (f *FileStore) Restore(ctx context.Context, backupID string) error {
	if err := f.checkActive(); err != nil {
		return err
	}

	backupPath := f.filePath + ".backup." + backupID
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	var fileData FileData
	if err := json.Unmarshal(data, &fileData); err != nil {
		return fmt.Errorf("failed to unmarshal backup data: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.tasks = make(map[int]models.Task)
	for _, task := range fileData.Tasks {
		f.tasks[task.ID] = task
		if task.ID > f.maxID {
			f.maxID = task.ID
		}
	}

	return f.save()
}

// Helper functions

func (f *FileStore) loadFromFile() error {
	data, err := os.ReadFile(f.filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var fileData FileData
	if err := json.Unmarshal(data, &fileData); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	f.tasks = make(map[int]models.Task)
	for _, task := range fileData.Tasks {
		f.tasks[task.ID] = task
		if task.ID > f.maxID {
			f.maxID = task.ID
		}
	}

	return nil
}

func (f *FileStore) save() error {
	data := f.prepareFileData()
	return f.saveToFile(data)
}

func (f *FileStore) saveToFile(data FileData) error {
	fileData, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := os.WriteFile(f.filePath, fileData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	f.lastSave = time.Now()
	return nil
}

func (f *FileStore) prepareFileData() FileData {
	tasks := make([]models.Task, 0, len(f.tasks))
	for _, task := range f.tasks {
		tasks = append(tasks, task)
	}

	return FileData{
		Metadata: FileMetadata{
			Version:     "1.0",
			LastUpdated: time.Now(),
			TaskCount:   len(tasks),
			MaxID:       f.maxID,
		},
		Tasks: tasks,
	}
}

func (f *FileStore) saveIfNeeded() error {
	if time.Since(f.lastSave) >= f.autoSaveTime {
		return f.save()
	}
	return nil
}

func (f *FileStore) autoSaveRoutine() {
	ticker := time.NewTicker(f.autoSaveTime)
	defer ticker.Stop()

	for {
		<-ticker.C
		if !f.isActive {
			return
		}

		f.mu.Lock()
		_ = f.save() // Ignore error as this is a background routine
		f.mu.Unlock()
	}
}

func (f *FileStore) checkActive() error {
	if !f.isActive {
		return storage.ErrStorageConnection
	}
	return nil
}

func (f *FileStore) ListTasks(ctx context.Context, filter *storage.Filter, sort *storage.SortOption, page *storage.Page) ([]models.Task, error) {
	if err := f.checkActive(); err != nil {
		return nil, err
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	// Convert map to slice for filtering and sorting
	var tasks []models.Task
	for _, task := range f.tasks {
		if f.matchesFilter(task, filter) {
			tasks = append(tasks, task)
		}
	}

	// Apply sorting
	if sort != nil {
		f.sortTasks(tasks, sort)
	}

	// Apply pagination
	if page != nil {
		return f.paginateTasks(tasks, page), nil
	}

	return tasks, nil
}

// SearchTasks performs a search across task fields
func (f *FileStore) SearchTasks(ctx context.Context, query string) ([]models.Task, error) {
	if err := f.checkActive(); err != nil {
		return nil, err
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	query = strings.ToLower(query)
	var results []models.Task

	for _, task := range f.tasks {
		if f.taskMatchesSearch(task, query) {
			results = append(results, task)
		}
	}

	return results, nil
}

// GetTasksByCategory returns tasks for a specific category
func (f *FileStore) GetTasksByCategory(ctx context.Context, category string) ([]models.Task, error) {
	return f.ListTasks(ctx, &storage.Filter{Category: category}, nil, nil)
}

// GetTasksByTag returns tasks that have a specific tag
func (f *FileStore) GetTasksByTag(ctx context.Context, tag string) ([]models.Task, error) {
	return f.ListTasks(ctx, &storage.Filter{Tags: []string{tag}}, nil, nil)
}

// GetTasksByStatus returns tasks with a specific status
func (f *FileStore) GetTasksByStatus(ctx context.Context, status models.TaskStatus) ([]models.Task, error) {
	return f.ListTasks(ctx, &storage.Filter{Status: &status}, nil, nil)
}

// GetOverdueTasks returns all overdue tasks
func (f *FileStore) GetOverdueTasks(ctx context.Context) ([]models.Task, error) {
	return f.ListTasks(ctx, &storage.Filter{IsOverdue: true}, nil, nil)
}

// GetUpcomingTasks returns tasks due in the next n days
func (f *FileStore) GetUpcomingTasks(ctx context.Context, days int) ([]models.Task, error) {
	dueDate := time.Now().AddDate(0, 0, days)
	return f.ListTasks(ctx, &storage.Filter{DueBefore: &dueDate}, nil, nil)
}

// CreateTasks creates multiple tasks in a batch
func (f *FileStore) CreateTasks(ctx context.Context, tasks []models.Task) error {
	if err := f.checkActive(); err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	for i := range tasks {
		f.maxID++
		tasks[i].ID = f.maxID
		tasks[i].CreatedAt = time.Now()
		tasks[i].UpdatedAt = time.Now()

		if err := tasks[i].Validate(); err != nil {
			return fmt.Errorf("validation failed for task %d: %w", i+1, err)
		}

		f.tasks[tasks[i].ID] = tasks[i]
	}

	return f.saveIfNeeded()
}

// DeleteTasks deletes multiple tasks in a batch
func (f *FileStore) DeleteTasks(ctx context.Context, ids []int) error {
	if err := f.checkActive(); err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	for _, id := range ids {
		if _, exists := f.tasks[id]; !exists {
			return fmt.Errorf("task %d: %w", id, storage.ErrTaskNotFound)
		}
		delete(f.tasks, id)
	}

	return f.saveIfNeeded()
}

// GetCategories returns all unique categories
func (f *FileStore) GetCategories(ctx context.Context) ([]string, error) {
	if err := f.checkActive(); err != nil {
		return nil, err
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	categories := make(map[string]struct{})
	for _, task := range f.tasks {
		if task.Category != "" {
			categories[task.Category] = struct{}{}
		}
	}

	result := make([]string, 0, len(categories))
	for category := range categories {
		result = append(result, category)
	}
	sort.Strings(result)
	return result, nil
}

// GetTags returns all unique tags
func (f *FileStore) GetTags(ctx context.Context) ([]string, error) {
	if err := f.checkActive(); err != nil {
		return nil, err
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	tags := make(map[string]struct{})
	for _, task := range f.tasks {
		for _, tag := range task.Tags {
			tags[tag] = struct{}{}
		}
	}

	result := make([]string, 0, len(tags))
	for tag := range tags {
		result = append(result, tag)
	}
	sort.Strings(result)
	return result, nil
}

// Export exports tasks in the specified format
func (f *FileStore) Export(ctx context.Context, format string) ([]byte, error) {
	if err := f.checkActive(); err != nil {
		return nil, err
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	data := f.prepareFileData()

	switch strings.ToLower(format) {
	case "json":
		return json.MarshalIndent(data, "", "    ")
	case "csv":
		return f.exportToCSV()
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// Import imports tasks from the specified format
func (f *FileStore) Import(ctx context.Context, data []byte, format string) error {
	if err := f.checkActive(); err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	switch strings.ToLower(format) {
	case "json":
		var fileData FileData
		if err := json.Unmarshal(data, &fileData); err != nil {
			return fmt.Errorf("failed to unmarshal JSON: %w", err)
		}
		return f.importFromFileData(fileData)
	case "csv":
		return f.importFromCSV(data)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// MarkTaskComplete marks a task as complete
func (f *FileStore) MarkTaskComplete(ctx context.Context, id int) error {
	task, err := f.GetTask(ctx, id)
	if err != nil {
		return err
	}

	task.Status = models.Completed
	task.CompletedAt = time.Now()
	return f.UpdateTask(ctx, task)
}

// MarkTaskIncomplete marks a task as incomplete
func (f *FileStore) MarkTaskIncomplete(ctx context.Context, id int) error {
	task, err := f.GetTask(ctx, id)
	if err != nil {
		return err
	}

	task.Status = models.NotStarted
	task.CompletedAt = time.Time{}
	return f.UpdateTask(ctx, task)
}

// GetProductivityStats returns productivity statistics
func (f *FileStore) GetProductivityStats(ctx context.Context, startDate, endDate time.Time) (map[string]interface{}, error) {
	if err := f.checkActive(); err != nil {
		return nil, err
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	stats := make(map[string]interface{})
	var totalTasks, completedTasks, overdueTasks int
	categoryStats := make(map[string]int)
	priorityStats := make(map[models.Priority]int)

	for _, task := range f.tasks {
		if task.CreatedAt.Before(startDate) || task.CreatedAt.After(endDate) {
			continue
		}

		totalTasks++
		if task.Status == models.Completed {
			completedTasks++
		}
		if task.IsOverdue() {
			overdueTasks++
		}

		categoryStats[task.Category]++
		priorityStats[task.Priority]++
	}

	stats["total_tasks"] = totalTasks
	stats["completed_tasks"] = completedTasks
	stats["completion_rate"] = float64(completedTasks) / float64(totalTasks) * 100
	stats["overdue_tasks"] = overdueTasks
	stats["category_distribution"] = categoryStats
	stats["priority_distribution"] = priorityStats

	return stats, nil
}

// Additional helper functions

func (f *FileStore) taskMatchesSearch(task models.Task, query string) bool {
	return strings.Contains(strings.ToLower(task.Name), query) ||
		strings.Contains(strings.ToLower(task.Description), query) ||
		strings.Contains(strings.ToLower(task.Category), query) ||
		f.containsTag(task.Tags, query)
}

func (f *FileStore) containsTag(tags []string, query string) bool {
	for _, tag := range tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}
	return false
}

func (f *FileStore) exportToCSV() ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header
	header := []string{"ID", "Name", "Description", "Status", "Priority", "Category",
		"Created At", "Due Date", "Completed At", "Tags"}
	if err := writer.Write(header); err != nil {
		return nil, err
	}

	// Write tasks
	for _, task := range f.tasks {
		record := []string{
			strconv.Itoa(task.ID),
			task.Name,
			task.Description,
			task.Status.String(),
			task.Priority.String(),
			task.Category,
			task.CreatedAt.Format(time.RFC3339),
			task.DueDate.Format(time.RFC3339),
			task.CompletedAt.Format(time.RFC3339),
			strings.Join(task.Tags, ";"),
		}
		if err := writer.Write(record); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	return buf.Bytes(), writer.Error()
}

func (f *FileStore) importFromCSV(data []byte) error {
	reader := csv.NewReader(bytes.NewReader(data))
	
	// Skip header
	if _, err := reader.Read(); err != nil {
		return err
	}

	records, err := reader.ReadAll()
	if err != nil {
		return err
	}

	for _, record := range records {
		if len(record) < 10 {
			continue
		}

		id, _ := strconv.Atoi(record[0])
		task := models.Task{
			ID:          id,
			Name:        record[1],
			Description: record[2],
			Category:    record[5],
			Tags:        strings.Split(record[9], ";"),
		}

		createdAt, _ := time.Parse(time.RFC3339, record[6])
		dueDate, _ := time.Parse(time.RFC3339, record[7])
		completedAt, _ := time.Parse(time.RFC3339, record[8])

		task.CreatedAt = createdAt
		task.DueDate = dueDate
		task.CompletedAt = completedAt

		f.tasks[task.ID] = task
		if task.ID > f.maxID {
			f.maxID = task.ID
		}
	}

	return f.save()
}

func (f *FileStore) importFromFileData(data FileData) error {
	for _, task := range data.Tasks {
		f.tasks[task.ID] = task
		if task.ID > f.maxID {
			f.maxID = task.ID
		}
	}
	return f.save()
}

func (f *FileStore) matchesFilter(task models.Task, filter *storage.Filter) bool {
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
 
	now := time.Now()
	if filter.DueBefore != nil && !task.DueDate.Before(*filter.DueBefore) {
	    return false
	}
 
	if filter.DueAfter != nil && !task.DueDate.After(*filter.DueAfter) {
	    return false
	}
 
	if filter.IsOverdue && (!task.DueDate.Before(now) || task.Status == models.Completed) {
	    return false
	}
 
	if filter.SearchTerm != "" {
	    searchTerm := strings.ToLower(filter.SearchTerm)
	    if !strings.Contains(strings.ToLower(task.Name), searchTerm) &&
		   !strings.Contains(strings.ToLower(task.Description), searchTerm) &&
		   !strings.Contains(strings.ToLower(task.Category), searchTerm) {
		   return false
	    }
	}
 
	return true
 }
 
 // sortTasks sorts the tasks based on the provided sorting options
 func (f *FileStore) sortTasks(tasks []models.Task, sort *storage.SortOption) {
	if sort == nil {
	    return
	}
 
	sort.Field = strings.ToLower(sort.Field)
 
	sorter := &taskSorter{
	    tasks: tasks,
	    less: func(i, j int) bool {
		   var result bool
		   switch sort.Field {
		   case "due_date":
			  if tasks[i].DueDate.IsZero() {
				 return false
			  }
			  if tasks[j].DueDate.IsZero() {
				 return true
			  }
			  result = tasks[i].DueDate.Before(tasks[j].DueDate)
		   case "priority":
			  result = tasks[i].Priority < tasks[j].Priority
		   case "created_at":
			  result = tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
		   case "updated_at":
			  result = tasks[i].UpdatedAt.Before(tasks[j].UpdatedAt)
		   case "completed_at":
			  if tasks[i].CompletedAt.IsZero() {
				 return false
			  }
			  if tasks[j].CompletedAt.IsZero() {
				 return true
			  }
			  result = tasks[i].CompletedAt.Before(tasks[j].CompletedAt)
		   case "status":
			  result = tasks[i].Status < tasks[j].Status
		   case "category":
			  result = tasks[i].Category < tasks[j].Category
		   case "name":
			  result = tasks[i].Name < tasks[j].Name
		   default: // default sort by ID
			  result = tasks[i].ID < tasks[j].ID
		   }
 
		   if !sort.Ascending {
			  return !result
		   }
		   return result
	    },
	}
 
	sort.Sort(sorter)
 }
 
 // paginateTasks returns a subset of tasks based on pagination parameters
 func (f *FileStore) paginateTasks(tasks []models.Task, page *storage.Page) []models.Task {
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
 
 // taskSorter implements sort.Interface for sorting tasks
 type taskSorter struct {
	tasks []models.Task
	less  func(i, j int) bool
 }
 
 func (s *taskSorter) Len() int {
	return len(s.tasks)
 }
 
 func (s *taskSorter) Less(i, j int) bool {
	return s.less(i, j)
 }
 
 func (s *taskSorter) Swap(i, j int) {
	s.tasks[i], s.tasks[j] = s.tasks[j], s.tasks[i]
 }
 
 // Additional helper function for calculating task statistics
 func (f *FileStore) calculateTaskStats(tasks []models.Task) map[string]interface{} {
	stats := make(map[string]interface{})
	now := time.Now()
 
	var (
	    totalTasks      int
	    completedTasks  int
	    overdueTasks    int
	    upcomingTasks   int
	    highPriorityTasks int
	)
 
	categoryStats := make(map[string]int)
	priorityStats := make(map[models.Priority]int)
	statusStats := make(map[models.TaskStatus]int)
 
	for _, task := range tasks {
	    totalTasks++
	    
	    // Status statistics
	    statusStats[task.Status]++
	    if task.Status == models.Completed {
		   completedTasks++
	    }
 
	    // Due date statistics
	    if !task.DueDate.IsZero() {
		   if task.DueDate.Before(now) && task.Status != models.Completed {
			  overdueTasks++
		   } else if task.DueDate.After(now) && task.DueDate.Before(now.AddDate(0, 0, 7)) {
			  upcomingTasks++
		   }
	    }
 
	    // Priority statistics
	    priorityStats[task.Priority]++
	    if task.Priority == models.High || task.Priority == models.Urgent {
		   highPriorityTasks++
	    }
 
	    // Category statistics
	    if task.Category != "" {
		   categoryStats[task.Category]++
	    }
	}
 
	// Calculate completion rate
	var completionRate float64
	if totalTasks > 0 {
	    completionRate = float64(completedTasks) / float64(totalTasks) * 100
	}
 
	stats["total_tasks"] = totalTasks
	stats["completed_tasks"] = completedTasks
	stats["completion_rate"] = completionRate
	stats["overdue_tasks"] = overdueTasks
	stats["upcoming_tasks"] = upcomingTasks
	stats["high_priority_tasks"] = highPriorityTasks
	stats["category_distribution"] = categoryStats
	stats["priority_distribution"] = priorityStats
	stats["status_distribution"] = statusStats
 
	return stats
 }