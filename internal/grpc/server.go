package grpc

import (
	"MainProject/internal/model"
	"MainProject/internal/repository"
	"context"
	"log"
	"time"

	pb "MainProject/api/proto"
)

type NotesServer struct {
	pb.UnimplementedNotesServiceServer
	repo *repository.Repository
}

func NewNotesServer(repo *repository.Repository) *NotesServer {
	return &NotesServer{
		repo: repo,
	}
}

// CreateNote создает новую заметку
func (s *NotesServer) CreateNote(ctx context.Context, req *pb.CreateNoteRequest) (*pb.NoteResponse, error) {
	log.Printf("Получен запрос на создание заметки: %s", req.Title)

	note := model.NewNote(req.Title, req.Content, req.Tags, req.IsPublic)
	note.IsGenerated = false

	id := s.repo.AddNote(note)
	note.SetID(id)

	// Сохраняем в CSV
	if err := s.repo.SaveNoteToCSV(note); err != nil {
		log.Printf("Ошибка при сохранении в CSV: %v", err)
	}

	return convertNoteToResponse(note), nil
}

// GetNote возвращает заметку по ID
func (s *NotesServer) GetNote(ctx context.Context, req *pb.GetNoteRequest) (*pb.NoteResponse, error) {
	log.Printf("Получен запрос на получение заметки с ID: %d", req.Id)

	note, found := s.repo.FindNoteById(req.Id)
	if !found {
		return nil, nil // В реальном проекте лучше вернуть ошибку
	}

	return convertNoteToResponse(note), nil
}

// ListNotes возвращает список всех заметок
func (s *NotesServer) ListNotes(ctx context.Context, req *pb.ListNotesRequest) (*pb.ListNotesResponse, error) {
	log.Println("Получен запрос на получение списка заметок")

	entities := s.repo.GetNotes()
	var notes []*pb.NoteResponse

	for _, entity := range entities {
		note := entity.(*model.Note)
		notes = append(notes, convertNoteToResponse(note))
	}

	return &pb.ListNotesResponse{
		Notes:      notes,
		TotalCount: int32(len(notes)),
	}, nil
}

// UpdateNote обновляет существующую заметку
func (s *NotesServer) UpdateNote(ctx context.Context, req *pb.UpdateNoteRequest) (*pb.NoteResponse, error) {
	log.Printf("Получен запрос на обновление заметки с ID: %d", req.Id)

	// Создаем обновленную заметку
	updated := model.NewNote(req.Title, req.Content, req.Tags, req.IsPublic)
	updated.SetID(req.Id)
	updated.IsGenerated = false

	if s.repo.UpdateNote(req.Id, updated) {
		return convertNoteToResponse(updated), nil
	}

	return nil, nil // Заметка не найдена
}

// DeleteNote удаляет заметку
func (s *NotesServer) DeleteNote(ctx context.Context, req *pb.DeleteNoteRequest) (*pb.DeleteNoteResponse, error) {
	log.Printf("Получен запрос на удаление заметки с ID: %d", req.Id)

	if s.repo.DeleteNote(req.Id) {
		// Удаляем из CSV
		if err := s.repo.DeleteNoteFromCSV(req.Id); err != nil {
			log.Printf("Ошибка при удалении из CSV: %v", err)
		}
		return &pb.DeleteNoteResponse{
			Success: true,
			Message: "Заметка успешно удалена",
		}, nil
	}

	return &pb.DeleteNoteResponse{
		Success: false,
		Message: "Заметка не найдена",
	}, nil
}

// Вспомогательная функция для конвертации модели в protobuf ответ
func convertNoteToResponse(note *model.Note) *pb.NoteResponse {
	return &pb.NoteResponse{
		Id:          note.ID().(int64),
		Title:       note.Title,
		Content:     note.Content,
		CreatedAt:   note.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   note.UpdatedAt.Format(time.RFC3339),
		Tags:        note.Tags,
		IsPublic:    note.IsPublic,
		IsGenerated: note.IsGenerated,
	}
}
