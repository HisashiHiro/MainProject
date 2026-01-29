package handlers

import (
	"MainProject/internal/model"
	"MainProject/internal/repository"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

func HandlePostItem(w http.ResponseWriter, r *http.Request, repo *repository.Repository) {
	var note model.Note
	if err := json.NewDecoder(r.Body).Decode(&note); err != nil {
		http.Error(w, "Неверный JSON", http.StatusBadRequest)
		return
	}

	if note.Title == "" || note.Content == "" {
		http.Error(w, "Поля 'title' и 'content' обязательны", http.StatusBadRequest)
		return
	}

	note.CreatedAt = time.Now()
	note.UpdatedAt = time.Now()
	note.IsGenerated = false

	id := repo.AddNote(&note)
	note.SetID(id)

	if err := repo.SaveNoteToCSV(&note); err != nil {
		http.Error(w, "Ошибка при сохранении в CSV", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(note)
}

func HandleGetItem(w http.ResponseWriter, r *http.Request, repo *repository.Repository, idStr string) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Неверный ID", http.StatusBadRequest)
		return
	}

	note, found := repo.FindNoteById(id)
	if !found {
		http.Error(w, "Заметка не найдена", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(note)
}

func HandleGetItems(w http.ResponseWriter, r *http.Request, repo *repository.Repository) {
	notes := repo.GetNotes()
	result := make([]model.Note, len(notes))
	for i, entity := range notes {
		result[i] = *entity.(*model.Note)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func HandlePutItem(w http.ResponseWriter, r *http.Request, repo *repository.Repository, idStr string) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Неверный ID", http.StatusBadRequest)
		return
	}

	var updated model.Note
	if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
		http.Error(w, "Неверный JSON", http.StatusBadRequest)
		return
	}

	if updated.Title == "" || updated.Content == "" {
		http.Error(w, "Поля 'title' и 'content' обязательны", http.StatusBadRequest)
		return
	}

	updated.SetID(id) // Используем метод
	updated.UpdatedAt = time.Now()

	if !repo.UpdateNote(id, &updated) {
		http.Error(w, "Заметка не найдена", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

func HandleDeleteItem(w http.ResponseWriter, r *http.Request, repo *repository.Repository, idStr string) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Неверный ID", http.StatusBadRequest)
		return
	}

	if !repo.DeleteNote(id) {
		http.Error(w, "Заметка не найдена", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content — успешно удалено
}
