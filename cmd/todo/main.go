package main

import (
	"log"

	"github.com/tiwariParth/go-todo-cli/internal/app"
	"github.com/tiwariParth/go-todo-cli/internal/cli"
	"github.com/tiwariParth/go-todo-cli/internal/storage/memory"
)

func main() {
    // Initialize storage
    store := memory.NewMemoryStore()
    
    // Initialize application
    todoApp := app.NewTodoApp(store)
    
    // Initialize CLI
    cli := cli.NewCLI(todoApp)
    
    // Run the application
    if err := cli.Run(); err != nil {
        log.Fatalf("Application error: %v", err)
    }
}