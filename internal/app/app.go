package app

import (
    "time"

    "github.com/tiwariParth/go-todo-cli/internal/models"
    "github.com/tiwariParth/go-todo-cli/internal/storage"
)

type TodoApp struct {
    store storage.Storage
}

func NewTodoApp(store storage.Storage) *TodoApp {
    return &TodoApp{store: store}
}

func (app *TodoApp) AddTask(name string, description string, priority models.Priority) error {
    task := &models.Task{
        Name:        name,
        Description: description,
        Priority:    priority,
        CreatedAt:   time.Now(),
        Completed:   false,
    }
    return app.store.CreateTask(task)
}

// Add other application logic methods