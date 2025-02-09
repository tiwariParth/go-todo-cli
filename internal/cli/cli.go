package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/tiwariParth/go-todo-cli/internal/app"
	"github.com/tiwariParth/go-todo-cli/internal/models"
)

type CLI struct {
	app    *app.TodoApp
	reader *bufio.Reader
	writer *tabwriter.Writer
}

// Command represents a CLI command
type Command struct {
	Name        string
	Description string
	Usage       string
	Action      func(args []string) error
}

func NewCLI(app *app.TodoApp) *CLI {
	return &CLI{
		app:    app,
		reader: bufio.NewReader(os.Stdin),
		writer: tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.TabIndent),
	}
}

func (c *CLI) Run() error {
	fmt.Println("Welcome to Todo CLI!")
	fmt.Println("Type 'help' for available commands")

	for {
		fmt.Print("\n> ")
		input, err := c.reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading input: %w", err)
		}

		input = strings.TrimSpace(input)
		args := strings.Fields(input)
		if len(args) == 0 {
			continue
		}

		cmd := args[0]
		cmdArgs := args[1:]

		if err := c.executeCommand(cmd, cmdArgs); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}
}

func (c *CLI) executeCommand(cmd string, args []string) error {
	ctx := context.Background()

	switch strings.ToLower(cmd) {
	case "help":
		return c.showHelp()
	case "add":
		return c.addTask(args)
	case "list":
		return c.listTasks(ctx, args)
	case "done":
		return c.markTaskComplete(ctx, args)
	case "undone":
		return c.markTaskIncomplete(ctx, args)
	case "delete":
		return c.deleteTask(ctx, args)
	case "update":
		return c.updateTask(ctx, args)
	case "show":
		return c.showTask(ctx, args)
	case "search":
		return c.searchTasks(ctx, args)
	case "stats":
		return c.showStats(ctx)
	case "categories":
		return c.listCategories(ctx)
	case "tags":
		return c.listTags(ctx)
	case "export":
		return c.exportTasks(ctx, args)
	case "import":
		return c.importTasks(ctx, args)
	case "backup":
		return c.backupTasks(ctx)
	case "restore":
		return c.restoreTasks(ctx, args)
	case "exit", "quit":
		fmt.Println("Goodbye!")
		os.Exit(0)
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}

	return nil
}

func (c *CLI) showHelp() error {
	commands := []Command{
		{"help", "Show this help message", "help", nil},
		{"add", "Add a new task", "add <name> [-d description] [-c category] [-p priority] [-due YYYY-MM-DD]", nil},
		{"list", "List tasks", "list [-c category] [-s status] [-p priority]", nil},
		{"done", "Mark task as complete", "done <task-id>", nil},
		{"undone", "Mark task as incomplete", "undone <task-id>", nil},
		{"delete", "Delete a task", "delete <task-id>", nil},
		{"update", "Update a task", "update <task-id> [-n name] [-d description] [-c category] [-p priority]", nil},
		{"show", "Show task details", "show <task-id>", nil},
		{"search", "Search tasks", "search <query>", nil},
		{"stats", "Show task statistics", "stats", nil},
		{"categories", "List all categories", "categories", nil},
		{"tags", "List all tags", "tags", nil},
		{"export", "Export tasks", "export [json|csv] <filename>", nil},
		{"import", "Import tasks", "import [json|csv] <filename>", nil},
		{"backup", "Backup tasks", "backup", nil},
		{"restore", "Restore from backup", "restore <backup-id>", nil},
		{"exit", "Exit the application", "exit", nil},
	}

	fmt.Println("\nAvailable Commands:")
	for _, cmd := range commands {
		fmt.Printf("  %-12s %s\n", cmd.Name, cmd.Description)
		fmt.Printf("    Usage: %s\n", cmd.Usage)
	}
	return nil
}

func (c *CLI) addTask(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task name is required")
	}

	task := &models.Task{
		Name:      args[0],
		CreatedAt: time.Now(),
		Status:    models.NotStarted,
		Priority:  models.Medium,
	}

	// Parse optional arguments
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-d", "--description":
			if i+1 < len(args) {
				task.Description = args[i+1]
				i++
			}
		case "-c", "--category":
			if i+1 < len(args) {
				task.Category = args[i+1]
				i++
			}
		case "-p", "--priority":
			if i+1 < len(args) {
				priority := strings.ToLower(args[i+1])
				switch priority {
				case "low":
					task.Priority = models.Low
				case "medium":
					task.Priority = models.Medium
				case "high":
					task.Priority = models.High
				case "urgent":
					task.Priority = models.Urgent
				default:
					return fmt.Errorf("invalid priority: %s", args[i+1])
				}
				i++
			}
		case "-due", "--due-date":
			if i+1 < len(args) {
				dueDate, err := time.Parse("2006-01-02", args[i+1])
				if err != nil {
					return fmt.Errorf("invalid due date format: %s", args[i+1])
				}
				task.DueDate = dueDate
				i++
			}
		case "-t", "--tags":
			if i+1 < len(args) {
				task.Tags = strings.Split(args[i+1], ",")
				i++
			}
		}
	}

	ctx := context.Background()
	if err := c.app.CreateTask(ctx, task); err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	fmt.Printf("Task created with ID: %d\n", task.ID)
	return nil
}

func (c *CLI) listTasks(ctx context.Context, args []string) error {
	var filter models.TaskFilter
	var sort models.SortOption

	// Parse arguments for filtering and sorting
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-c", "--category":
			if i+1 < len(args) {
				filter.Category = args[i+1]
				i++
			}
		case "-s", "--status":
			if i+1 < len(args) {
				status := strings.ToLower(args[i+1])
				switch status {
				case "not-started":
					filter.Status = models.NotStarted
				case "in-progress":
					filter.Status = models.InProgress
				case "completed":
					filter.Status = models.Completed
				default:
					return fmt.Errorf("invalid status: %s", args[i+1])
				}
				i++
			}
		case "-p", "--priority":
			if i+1 < len(args) {
				priority := strings.ToLower(args[i+1])
				switch priority {
				case "low":
					filter.Priority = models.Low
				case "medium":
					filter.Priority = models.Medium
				case "high":
					filter.Priority = models.High
				case "urgent":
					filter.Priority = models.Urgent
				default:
					return fmt.Errorf("invalid priority: %s", args[i+1])
				}
				i++
			}
		case "--sort":
			if i+1 < len(args) {
				sort.Field = args[i+1]
				i++
			}
		case "--asc":
			sort.Ascending = true
		case "--desc":
			sort.Ascending = false
		}
	}

	tasks, err := c.app.ListTasks(ctx, &filter, &sort)
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Println("No tasks found")
		return nil
	}

	c.printTasks(tasks)
	return nil
}

func (c *CLI) printTasks(tasks []models.Task) {
	w := c.writer
	fmt.Fprintln(w, "ID\tName\tStatus\tPriority\tCategory\tDue Date\tTags")
	fmt.Fprintln(w, "--\t----\t------\t--------\t--------\t--------\t----")

	for _, task := range tasks {
		dueDate := "-"
		if !task.DueDate.IsZero() {
			dueDate = task.DueDate.Format("2006-01-02")
		}

		tags := "-"
		if len(task.Tags) > 0 {
			tags = strings.Join(task.Tags, ", ")
		}

		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
			task.ID,
			task.Name,
			task.Status,
			task.Priority,
			task.Category,
			dueDate,
			tags,
		)
	}
	w.Flush()
}

// ... (previous code remains the same)

func (c *CLI) markTaskComplete(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task ID is required")
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid task ID: %s", args[0])
	}

	if err := c.app.MarkTaskComplete(ctx, id); err != nil {
		return fmt.Errorf("failed to mark task as complete: %w", err)
	}

	fmt.Printf("Task %d marked as complete\n", id)
	return nil
}

func (c *CLI) markTaskIncomplete(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task ID is required")
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid task ID: %s", args[0])
	}

	if err := c.app.MarkTaskIncomplete(ctx, id); err != nil {
		return fmt.Errorf("failed to mark task as incomplete: %w", err)
	}

	fmt.Printf("Task %d marked as incomplete\n", id)
	return nil
}

func (c *CLI) deleteTask(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task ID is required")
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid task ID: %s", args[0])
	}

	// Ask for confirmation
	fmt.Printf("Are you sure you want to delete task %d? (y/N): ", id)
	confirmation, err := c.reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}

	if strings.ToLower(strings.TrimSpace(confirmation)) != "y" {
		fmt.Println("Operation cancelled")
		return nil
	}

	if err := c.app.DeleteTask(ctx, id); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	fmt.Printf("Task %d deleted\n", id)
	return nil
}

func (c *CLI) updateTask(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("task ID and at least one field to update are required")
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid task ID: %s", args[0])
	}

	task, err := c.app.GetTask(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Parse update fields
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-n", "--name":
			if i+1 < len(args) {
				task.Name = args[i+1]
				i++
			}
		case "-d", "--description":
			if i+1 < len(args) {
				task.Description = args[i+1]
				i++
			}
		case "-c", "--category":
			if i+1 < len(args) {
				task.Category = args[i+1]
				i++
			}
		case "-p", "--priority":
			if i+1 < len(args) {
				priority := strings.ToLower(args[i+1])
				switch priority {
				case "low":
					task.Priority = models.Low
				case "medium":
					task.Priority = models.Medium
				case "high":
					task.Priority = models.High
				case "urgent":
					task.Priority = models.Urgent
				default:
					return fmt.Errorf("invalid priority: %s", args[i+1])
				}
				i++
			}
		case "-s", "--status":
			if i+1 < len(args) {
				status := strings.ToLower(args[i+1])
				switch status {
				case "not-started":
					task.Status = models.NotStarted
				case "in-progress":
					task.Status = models.InProgress
				case "completed":
					task.Status = models.Completed
				default:
					return fmt.Errorf("invalid status: %s", args[i+1])
				}
				i++
			}
		case "-due", "--due-date":
			if i+1 < len(args) {
				dueDate, err := time.Parse("2006-01-02", args[i+1])
				if err != nil {
					return fmt.Errorf("invalid due date format: %s", args[i+1])
				}
				task.DueDate = dueDate
				i++
			}
		case "-t", "--tags":
			if i+1 < len(args) {
				task.Tags = strings.Split(args[i+1], ",")
				i++
			}
		}
	}

	if err := c.app.UpdateTask(ctx, task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	fmt.Printf("Task %d updated successfully\n", id)
	return nil
}

func (c *CLI) showTask(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task ID is required")
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid task ID: %s", args[0])
	}

	task, err := c.app.GetTask(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	c.printTaskDetails(task)
	return nil
}

func (c *CLI) searchTasks(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("search query is required")
	}

	query := strings.Join(args, " ")
	tasks, err := c.app.SearchTasks(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to search tasks: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Println("No tasks found matching your search")
		return nil
	}

	c.printTasks(tasks)
	return nil
}

func (c *CLI) showStats(ctx context.Context) error {
	stats, err := c.app.GetProductivityStats(ctx, time.Now().AddDate(0, -1, 0), time.Now())
	if err != nil {
		return fmt.Errorf("failed to get statistics: %w", err)
	}

	w := c.writer
	fmt.Fprintln(w, "\nTask Statistics:")
	fmt.Fprintln(w, "---------------")
	fmt.Fprintf(w, "Total Tasks:\t%d\n", stats["total_tasks"])
	fmt.Fprintf(w, "Completed Tasks:\t%d\n", stats["completed_tasks"])
	fmt.Fprintf(w, "Completion Rate:\t%.1f%%\n", stats["completion_rate"])
	fmt.Fprintf(w, "Overdue Tasks:\t%d\n", stats["overdue_tasks"])
	fmt.Fprintf(w, "High Priority Tasks:\t%d\n", stats["high_priority_tasks"])

	fmt.Fprintln(w, "\nCategory Distribution:")
	categoryStats := stats["category_distribution"].(map[string]int)
	for category, count := range categoryStats {
		fmt.Fprintf(w, "%s:\t%d\n", category, count)
	}

	w.Flush()
	return nil
}

func (c *CLI) listCategories(ctx context.Context) error {
	categories, err := c.app.GetCategories(ctx)
	if err != nil {
		return fmt.Errorf("failed to get categories: %w", err)
	}

	if len(categories) == 0 {
		fmt.Println("No categories found")
		return nil
	}

	fmt.Println("\nCategories:")
	for _, category := range categories {
		fmt.Printf("- %s\n", category)
	}
	return nil
}

func (c *CLI) listTags(ctx context.Context) error {
	tags, err := c.app.GetTags(ctx)
	if err != nil {
		return fmt.Errorf("failed to get tags: %w", err)
	}

	if len(tags) == 0 {
		fmt.Println("No tags found")
		return nil
	}

	fmt.Println("\nTags:")
	for _, tag := range tags {
		fmt.Printf("- %s\n", tag)
	}
	return nil
}

func (c *CLI) exportTasks(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("format and filename are required")
	}

	format := strings.ToLower(args[0])
	filename := args[1]

	data, err := c.app.ExportTasks(ctx, format)
	if err != nil {
		return fmt.Errorf("failed to export tasks: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("Tasks exported to %s\n", filename)
	return nil
}

func (c *CLI) importTasks(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("format and filename are required")
	}

	format := strings.ToLower(args[0])
	filename := args[1]

	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	if err := c.app.ImportTasks(ctx, data, format); err != nil {
		return fmt.Errorf("failed to import tasks: %w", err)
	}

	fmt.Println("Tasks imported successfully")
	return nil
}

func (c *CLI) backupTasks(ctx context.Context) error {
	if err := c.app.Backup(ctx); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	fmt.Println("Backup created successfully")
	return nil
}

func (c *CLI) restoreTasks(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("backup ID is required")
	}

	if err := c.app.Restore(ctx, args[0]); err != nil {
		return fmt.Errorf("failed to restore from backup: %w", err)
	}

	fmt.Println("Tasks restored successfully")
	return nil
}

func (c *CLI) printTaskDetails(task *models.Task) {
	w := c.writer
	fmt.Fprintln(w, "\nTask Details:")
	fmt.Fprintln(w, "-------------")
	fmt.Fprintf(w, "ID:\t%d\n", task.ID)
	fmt.Fprintf(w, "Name:\t%s\n", task.Name)
	fmt.Fprintf(w, "Description:\t%s\n", task.Description)
	fmt.Fprintf(w, "Status:\t%s\n", task.Status)
	fmt.Fprintf(w, "Priority:\t%s\n", task.Priority)
	fmt.Fprintf(w, "Category:\t%s\n", task.Category)
	
	if !task.DueDate.IsZero() {
		fmt.Fprintf(w, "Due Date:\t%s\n", task.DueDate.Format("2006-01-02"))
	}
	
	if !task.CreatedAt.IsZero() {
		fmt.Fprintf(w, "Created:\t%s\n", task.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	
	if !task.CompletedAt.IsZero() {
		fmt.Fprintf(w, "Completed:\t%s\n", task.CompletedAt.Format("2006-01-02 15:04:05"))
	}
	
	if len(task.Tags) > 0 {
		fmt.Fprintf(w, "Tags:\t%s\n", strings.Join(task.Tags, ", "))
	}

	w.Flush()
}