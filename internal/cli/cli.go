package cli

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tiwariParth/go-todo-cli/internal/task"
)

// CLI represents the command-line interface.
type CLI struct {
	Store *task.TaskStore
}

// NewCLI initializes a new CLI.
func NewCLI(store *task.TaskStore) *CLI {
	return &CLI{Store: store}
}

// Run executes the CLI based on the provided arguments.
func (c *CLI) Run(args []string) error {
	if len(args) < 1 {
		return errors.New("no command provided")
	}

	switch args[0] {
	case "add":
		if len(args) < 2 {
			return errors.New("missing task name")
		}
		name := strings.Join(args[1:], " ")
		priority := "" // Default priority
		dueDate := time.Time{} // No due date by default
		task, err := c.Store.AddTask(name, priority, dueDate)
		if err != nil {
			return fmt.Errorf("failed to add task: %w", err)
		}
		fmt.Printf("Added task: %s (ID: %d)\n", Bold(task.Name), task.ID)

	case "list":
		if len(c.Store.Tasks) == 0 {
			fmt.Println("No tasks found.")
			return nil
		}
		for id, task := range c.Store.Tasks {
			status := Red("Pending")
			if task.Completed {
				status = Green("Completed")
			}
			fmt.Printf("%d. %s - %s\n", id, Bold(task.Name), status)
		}

	case "complete":
		if len(args) < 2 {
			return errors.New("missing task ID")
		}
		id, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid task ID: %w", err)
		}
		task, exists := c.Store.Tasks[id]
		if !exists {
			return fmt.Errorf("task with ID %d not found", id)
		}
		task.MarkComplete()
		c.Store.Tasks[id] = task
		fmt.Printf("Marked task %d as completed: %s\n", id, Bold(task.Name))

	case "delete":
		if len(args) < 2 {
			return errors.New("missing task ID")
		}
		id, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid task ID: %w", err)
		}
		if _, exists := c.Store.Tasks[id]; !exists {
			return fmt.Errorf("task with ID %d not found", id)
		}
		delete(c.Store.Tasks, id)
		fmt.Printf("Deleted task %d\n", id)

	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}

	return nil
}