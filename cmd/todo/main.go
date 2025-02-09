package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Task struct {
	ID        int
	Name      string
	Completed bool
}

var tasks []Task

func addTask(name string) {
	task := Task{
		ID:        len(tasks) + 1,
		Name:      name,
		Completed: false,
	}
	tasks = append(tasks, task)
	fmt.Printf("Added task: %s\n", name)
}

func listTasks() {
	if len(tasks) == 0 {
		fmt.Println("No tasks available.")
		return
	}

	for _, task := range tasks {
		status := "Incomplete"
		if task.Completed {
			status = "Completed"
		}
		fmt.Printf("%d. %s [%s]\n", task.ID, task.Name, status)
	}
}

func deleteTask(id int) {
	for i, task := range tasks {
		if task.ID == id {
			tasks = append(tasks[:i], tasks[i+1:]...)
			fmt.Printf("Task %d deleted.\n", id)
			return
		}
	}
	fmt.Printf("Task with ID %d not found.\n", id)
}

func main() {
	fmt.Println("Welcome to Go Todo CLI!")
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		command, _ := reader.ReadString('\n')
		command = strings.TrimSpace(command) // Remove newline and spaces

		switch command {
		case "add":
			fmt.Print("Enter task name: ")
			taskName, _ := reader.ReadString('\n')
			taskName = strings.TrimSpace(taskName)
			if taskName == "" {
				fmt.Println("Task name cannot be empty.")
				continue
			}
			addTask(taskName)

		case "list":
			listTasks()

		case "delete":
			fmt.Print("Enter task ID to delete: ")
			var taskID int
			_, err := fmt.Scanln(&taskID)
			if err != nil {
				fmt.Println("Invalid input. Please enter a valid task ID.")
				continue
			}
			deleteTask(taskID)

		case "exit":
			fmt.Println("Goodbye!")
			return

		default:
			fmt.Println("Unknown command. Available commands: add, list, delete, exit")
		}
	}
}