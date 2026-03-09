package grpc

import (
	"MainProject/internal/model"
	"MainProject/internal/service"
	"context"
	"log"
	"time"

	pb "MainProject/api/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NotesServer реализует gRPC‑интерфейс работы с заметками,
// опираясь на доменный сервис NoteService.
type NotesServer struct {
	pb.UnimplementedNotesServiceServer
	svc service.NoteService
}

// NewNotesServer создаёт новый gRPC‑сервер заметок поверх NoteService.
func NewNotesServer(svc service.NoteService) *NotesServer {
	return &NotesServer{
		svc: svc,
	}
}

// CreateNote создает новую заметку
func (s *NotesServer) CreateNote(ctx context.Context, req *pb.CreateNoteRequest) (*pb.NoteResponse, error) {
	log.Printf("Получен запрос на создание заметки: %s", req.Title)

	note, err := s.svc.CreateNote(ctx, service.CreateNoteInput{
		Title:    req.Title,
		Content:  req.Content,
		Tags:     req.Tags,
		IsPublic: req.IsPublic,
	})
	if err != nil {
		log.Printf("Ошибка при создании заметки: %v", err)
		if err == service.ErrInvalidNote {
			return nil, status.Errorf(codes.InvalidArgument, "invalid note: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "failed to create note")
	}

	return convertNoteToResponse(note), nil
}

// GetNote возвращает заметку по ID
func (s *NotesServer) GetNote(ctx context.Context, req *pb.GetNoteRequest) (*pb.NoteResponse, error) {
	log.Printf("Получен запрос на получение заметки с ID: %d", req.Id)

	note, err := s.svc.GetNote(ctx, req.Id)
	if err != nil {
		if err == service.ErrNoteNotFound {
			return nil, status.Errorf(codes.NotFound, "note %d not found", req.Id)
		}
		log.Printf("Ошибка при получении заметки: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to get note")
	}

	return convertNoteToResponse(note), nil
}

// ListNotes возвращает список всех заметок
func (s *NotesServer) ListNotes(ctx context.Context, req *pb.ListNotesRequest) (*pb.ListNotesResponse, error) {
	log.Println("Получен запрос на получение списка заметок")

	entities, err := s.svc.ListNotes(ctx)
	if err != nil {
		log.Printf("Ошибка при получении списка заметок: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to list notes")
	}

	var notes []*pb.NoteResponse

	for _, note := range entities {
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

	updated, err := s.svc.UpdateNote(ctx, req.Id, service.UpdateNoteInput{
		Title:    req.Title,
		Content:  req.Content,
		Tags:     req.Tags,
		IsPublic: req.IsPublic,
	})
	if err != nil {
		if err == service.ErrNoteNotFound {
			return nil, status.Errorf(codes.NotFound, "note %d not found", req.Id)
		}
		if err == service.ErrInvalidNote {
			return nil, status.Errorf(codes.InvalidArgument, "invalid note: %v", err)
		}
		log.Printf("Ошибка при обновлении заметки: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to update note")
	}

	return convertNoteToResponse(updated), nil
}

// DeleteNote удаляет заметку
func (s *NotesServer) DeleteNote(ctx context.Context, req *pb.DeleteNoteRequest) (*pb.DeleteNoteResponse, error) {
	log.Printf("Получен запрос на удаление заметки с ID: %d", req.Id)

	err := s.svc.DeleteNote(ctx, req.Id)
	if err == nil {
		return &pb.DeleteNoteResponse{
			Success: true,
			Message: "Заметка успешно удалена",
		}, nil
	} else if err == service.ErrNoteNotFound {
		return nil, status.Errorf(codes.NotFound, "note %d not found", req.Id)
	}

	log.Printf("Ошибка при удалении заметки: %v", err)
	return nil, status.Errorf(codes.Internal, "failed to delete note")
}

// Вспомогательная функция для конвертации модели в protobuf ответ
func convertNoteToResponse(note *model.Note) *pb.NoteResponse {
	return &pb.NoteResponse{
		Id:          note.ID,
		Title:       note.Title,
		Content:     note.Content,
		CreatedAt:   note.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   note.UpdatedAt.Format(time.RFC3339),
		Tags:        note.Tags,
		IsPublic:    note.IsPublic,
		IsGenerated: note.IsGenerated,
	}
}
