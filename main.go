package main

import (
	"encoding/json"
	"github.com/google/uuid"
	"log"
	"net/http"
	"strings"
	"time"
)

type Task struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	ArchivedAt time.Time `json:"archived_at"`
}

var tasks = make(map[string]Task)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/tasks", handlerTasks)
	mux.HandleFunc("/tasks/", handlerTaskByID)

	log.Println("Server start")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Printf("Server error %v\n", err)
	}
}

func handlerTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getTasks(w, r)
	case http.MethodPost:
		addTask(w, r)
	}
}

func getTasks(w http.ResponseWriter, r *http.Request) {
	var tasksList = []Task{}
	for _, task := range tasks {
		tasksList = append(tasksList, task)
	}
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(tasksList)
}

func addTask(w http.ResponseWriter, r *http.Request) {
	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, "JSON error", http.StatusBadRequest)
		return
	}
	task.ID = uuid.New().String()
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	task.Status = "new"

	tasks[task.ID] = task

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

func handlerTaskByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/tasks/")
	task, ok := tasks[id]
	if !ok {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}
	switch r.Method {
	case http.MethodGet:
		json.NewEncoder(w).Encode(task)
	case http.MethodDelete:
		archiveTask(w, &task)
	case http.MethodPut:
		updateTask(w, r, &task)
	}
}

func archiveTask(w http.ResponseWriter, task *Task) {
	task.Status = "archived"
	task.ArchivedAt = time.Now()
	task.UpdatedAt = time.Now()
	tasks[task.ID] = *task

	w.WriteHeader(http.StatusNoContent)
}

func updateTask(w http.ResponseWriter, r *http.Request, task *Task) {
	var updateData = make(map[string]interface{})
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		http.Error(w, "Bad type", http.StatusBadRequest)
	}
	if title, ok := updateData["title"].(string); ok {
		task.Title = title
	}
	if status, ok := updateData["status"].(string); ok {
		task.Status = status
	}
	task.UpdatedAt = time.Now()

	tasks[task.ID] = *task

	json.NewEncoder(w).Encode(task)
	w.Header().Set("Content-Type", "application/json")
}
