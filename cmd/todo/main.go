package main

import (
	"fmt"
	"os"

	"github.com/tiwariParth/go-todo-cli/internal/cli"
	"github.com/tiwariParth/go-todo-cli/internal/task"
)

const dataFile = "tasks.json"

func main() {
	store := task.NewTaskStore()

	// Load tasks from file (if it exists)
	if err := store.LoadFromFile(dataFile); err != nil {
		fmt.Printf("Warning: Failed to load tasks: %v\n", err)
	}

	// Initialize CLI
	app := cli.NewCLI(store)

	// Run CLI with command-line arguments
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Save tasks to file
	if err := store.SaveToFile(dataFile); err != nil {
		fmt.Printf("Error: Failed to save tasks: %v\n", err)
		os.Exit(1)
	}
}