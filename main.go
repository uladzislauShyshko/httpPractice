package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

/*
Структура задачи
Исполнитель задач базы данных из конфига
Кастомные ошибки
Чистая архитектура (обработчики низшего уровня не могут знать об http)
Использование Mutex в структуре мапы как бд
*/

func main() {
	mux := http.NewServeMux()

	server := Server{DB: nil}

	mux.HandleFunc("/tasks", server.handleTasks)
	mux.HandleFunc("/tasks/", server.handleTaskByID)

	log.Println("server has started")

	if err := http.ListenAndServe("localhost:8080", mux); err != nil {
		log.Printf("Server error: %v\n", err)
	}
}

type Task struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	ArchivedAt time.Time `json:"archived_at"`
}

type Saver interface {
	AddTasks(data []Task) error
	GetTasks() ([]Task, error)
	GetTask(ID string) (*Task, error)
	UpdateTask(data map[string]interface{}, ID string) (*Task, error)
	ArchiveTask(ID string) error
}

type Server struct {
	DB Saver
}

var (
	ErrNotFound = errors.New("not found")
	ErrIsExist  = errors.New("this data is already exists")
)

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.GetTasks(w)
	case http.MethodPost:
		s.AddTasks(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) GetTasks(w http.ResponseWriter) {
	tasks, err := s.DB.GetTasks()

	if err != nil {
		http.Error(w, fmt.Sprintf("DB error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func (s *Server) AddTasks(w http.ResponseWriter, r *http.Request) {
	var tasks []Task

	if err := json.NewDecoder(r.Body).Decode(&tasks); err != nil {
		http.Error(w, fmt.Sprintf("JSON error: %v", err), http.StatusBadRequest)
		return
	}

	if err := s.DB.AddTasks(tasks); err != nil {
		if errors.Is(err, ErrIsExist) {
			http.Error(w, ErrIsExist.Error(), http.StatusBadRequest)
			return
		} else {
			http.Error(w, fmt.Sprintf("DB error: %v", err), http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func (s *Server) handleTaskByID(w http.ResponseWriter, r *http.Request) {
	ID := strings.TrimPrefix(r.URL.Path, "/tasks/")

	if ID == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.GetTask(w, ID)
	case http.MethodPut:
		s.UpdateTask(w, r, ID)
	case http.MethodDelete:
		s.ArchiveTask(w, ID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) GetTask(w http.ResponseWriter, ID string) {
	task, err := s.DB.GetTask(ID)

	if errors.Is(err, ErrNotFound) {
		http.Error(w, ErrNotFound.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(*task)
}

func (s *Server) UpdateTask(w http.ResponseWriter, r *http.Request, ID string) {
	var data = make(map[string]interface{})
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, fmt.Sprintf("DB error: %v", err), http.StatusInternalServerError)
	}

	task, err := s.DB.UpdateTask(data, ID)

	if errors.Is(err, ErrNotFound) {
		http.Error(w, ErrNotFound.Error(), http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, fmt.Sprintf("DB error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(*task)
}

func (s *Server) ArchiveTask(w http.ResponseWriter, ID string) {
	err := s.DB.ArchiveTask(ID)

	if errors.Is(err, ErrNotFound) {
		http.Error(w, ErrNotFound.Error(), http.StatusNotFound)
	} else if err != nil {
		http.Error(w, fmt.Sprintf("DB error: %v", err), http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusNoContent)
}

type MapDB struct {
	data map[string]*Task
	mx   sync.RWMutex
}

func (db *MapDB) AddTasks(newData []Task) error {
	defer db.mx.Unlock()
	db.mx.Lock()
	for _, task := range newData {
		task.CreatedAt = time.Now()
		task.UpdatedAt = time.Now()
		task.Status = "created"
		db.data[task.ID] = &task
	}
	return nil
}

func (db *MapDB) GetTasks() ([]Task, error) {
	var tasks []Task

	for _, task := range db.data {
		tasks = append(tasks, *task)
	}
	return tasks, nil
}

func (db *MapDB) GetTask(ID string) (*Task, error) {
	task, ok := db.data[ID]
	if !ok {
		return nil, ErrNotFound
	}

	return task, nil
}

func (db *MapDB) UpdateTask(data map[string]interface{}, ID string) (*Task, error) {
	task, err := db.GetTask(ID)

	if err != nil {
		return nil, err
	}

	title, ok := data["title"].(string)
	if ok {
		task.Title = title
	}

	status, ok := data["status"].(string)
	if ok {
		task.Status = status
	}
	task.UpdatedAt = time.Now()

	db.mx.RLock()
	db.data[ID] = task
	db.mx.RUnlock()

	return task, nil
}

func (db *MapDB) ArchiveTask(ID string) error {
	task, err := db.GetTask(ID)
	if err != nil {
		return err
	}
	task.ArchivedAt = time.Now()
	task.Status = "archived"

	db.mx.RLock()
	db.data[ID] = task
	db.mx.RUnlock()

	return nil
}
