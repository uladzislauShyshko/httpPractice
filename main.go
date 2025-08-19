package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

/*
get на все задачи
на одну задачу
post задачи
delete (арчайв)
put задачи

структура с методами работы с бд
интерфейс определяющий методы БД

*/

type Task struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	ArchivedAt time.Time `json:"archived_at"`
}

func main() {
	mux := http.NewServeMux()
	server := Server{}

	mux.HandleFunc("/tasks", server.handlerTasks)
	mux.HandleFunc("/tasks/", server.handlerTaskByID)

	log.Println("Server has started")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Printf("Server error: %v", err)
		return
	}
}

type Saver interface {
	AddTasks(data []Task) error
	ArchiveTask(ID string) error
	GetAllTasks() ([]Task, error)
	GetTask(ID string) (*Task, error)
	UpdateTask(data map[string]interface{}, ID string) (*Task, error)
}

type Server struct {
	DB Saver
}

// GLOBAL HANDLER
func (s Server) handlerTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.GetAllTasks(w)
	case http.MethodPost:
		s.AddTasks(w, r)
	default:
		http.Error(w, "Method is not allowed", http.StatusMethodNotAllowed)
	}
}

// METHODS:

func (s Server) GetAllTasks(w http.ResponseWriter) {
	tasks, err := s.DB.GetAllTasks()
	if err != nil {
		http.Error(w, fmt.Sprintf("DB error: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func (s Server) AddTasks(w http.ResponseWriter, r *http.Request) {
	var tasks []Task
	json.NewDecoder(r.Body).Decode(&tasks)

	err := s.DB.AddTasks(tasks)
	if err != nil {
		http.Error(w, fmt.Sprintf("DB error: %v", err), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": fmt.Sprintf("Added tasks %d", len(tasks)),
		"count":   len(tasks),
	})
}

// HANDLER BY ID
func (s Server) handlerTaskByID(w http.ResponseWriter, r *http.Request) {
	ID := strings.TrimPrefix(r.URL.Path, "/tasks/")

	if ID == "" {
		http.Error(w, "ID is required", http.StatusNotFound)
	}

	switch r.Method {
	case http.MethodGet:
		s.GetTask(w, ID)
	case http.MethodPut:
		s.UpdateTask(w, r, ID)
	case http.MethodDelete:
		s.ArchiveTask(w, ID)
	}
}

// handlerTaskByID Methods:

func (s Server) GetTask(w http.ResponseWriter, ID string) {
	task, err := s.DB.GetTask(ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("DB error: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(*task)
}

func (s Server) UpdateTask(w http.ResponseWriter, r *http.Request, ID string) {
	var newData = make(map[string]interface{}, 64)
	json.NewDecoder(r.Body).Decode(&newData)

	task, err := s.DB.UpdateTask(newData, ID)

	if err != nil {
		http.Error(w, fmt.Sprintf("DB error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(*task)

}

func (s Server) ArchiveTask(w http.ResponseWriter, ID string) {
	err := s.DB.ArchiveTask(ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("DB error: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}

type MapDB struct {
	Data map[string]Task
}

func (db MapDB) GetAllTasks() ([]Task, error) {
	var tasks []Task
	for _, task := range db.Data {
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (db MapDB) AddTasks(tasks []Task) error {
	for _, task := range tasks {
		task.CreatedAt = time.Now()
		task.UpdatedAt = time.Now()
		task.Status = "recently-created"
		db.Data[task.ID] = task
	}
	return nil
}

func (db MapDB) GetTask(ID string) (*Task, error) {
	task, ok := db.Data[ID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return &task, nil
}

func (db MapDB) UpdateTask(data map[string]interface{}, ID string) (*Task, error) {
	var task Task
	task, ok := db.Data[ID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}

	title, ok := data["title"].(string)
	if ok {
		task.Title = title
	}

	status, ok := data["status"].(string)
	if ok {
		task.Status = status
	}

	task.Status = "recently-updated"
	task.UpdatedAt = time.Now()
	db.Data[ID] = task
	return &task, nil
}

func (db MapDB) ArchiveTask(ID string) error {
	task, ok := db.Data[ID]
	if !ok {
		return fmt.Errorf("not found")
	}
	task.ArchivedAt = time.Now()
	task.UpdatedAt = time.Now()
	task.Status = "archived"
	db.Data[ID] = task

	return nil
}
