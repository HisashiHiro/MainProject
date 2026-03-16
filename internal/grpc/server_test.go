package grpc_test

import (
	"context"
	"net"
	"testing"
	"time"

	pb "MainProject/api/proto"
	grpcimpl "MainProject/internal/grpc"
	"MainProject/internal/model"
	"MainProject/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type mockNoteServiceForGRPC struct {
	createNoteFunc func(ctx context.Context, userID int64, input service.CreateNoteInput) (*model.Note, error)
	getNoteFunc    func(ctx context.Context, userID int64, id int64) (*model.Note, error)
	listNotesFunc  func(ctx context.Context, userID int64) ([]*model.Note, error)
	updateNoteFunc func(ctx context.Context, userID int64, id int64, input service.UpdateNoteInput) (*model.Note, error)
	deleteNoteFunc func(ctx context.Context, userID int64, id int64) error
}

func (m *mockNoteServiceForGRPC) CreateNote(ctx context.Context, userID int64, input service.CreateNoteInput) (*model.Note, error) {
	return m.createNoteFunc(ctx, userID, input)
}
func (m *mockNoteServiceForGRPC) GetNote(ctx context.Context, userID int64, id int64) (*model.Note, error) {
	return m.getNoteFunc(ctx, userID, id)
}
func (m *mockNoteServiceForGRPC) ListNotes(ctx context.Context, userID int64) ([]*model.Note, error) {
	return m.listNotesFunc(ctx, userID)
}
func (m *mockNoteServiceForGRPC) UpdateNote(ctx context.Context, userID int64, id int64, input service.UpdateNoteInput) (*model.Note, error) {
	return m.updateNoteFunc(ctx, userID, id, input)
}
func (m *mockNoteServiceForGRPC) DeleteNote(ctx context.Context, userID int64, id int64) error {
	return m.deleteNoteFunc(ctx, userID, id)
}

func TestGRPCCreateNote(t *testing.T) {
	// Мок сервиса
	mockSvc := &mockNoteServiceForGRPC{
		createNoteFunc: func(ctx context.Context, userID int64, input service.CreateNoteInput) (*model.Note, error) {
			assert.Equal(t, int64(42), userID)
			assert.Equal(t, "gRPC Title", input.Title)
			assert.Equal(t, "gRPC Content", input.Content)
			return &model.Note{
				ID:        100,
				UserID:    userID,
				Title:     input.Title,
				Content:   input.Content,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}

	// Создаём gRPC сервер
	server := grpc.NewServer()
	notesServer := grpcimpl.NewNotesServer(mockSvc)
	pb.RegisterNotesServiceServer(server, notesServer)

	// Буферизованный listener для in-memory соединения
	listener := bufconn.Listen(1024 * 1024)
	go func() {
		if err := server.Serve(listener); err != nil {
			t.Errorf("Server exited with error: %v", err)
		}
	}()
	defer server.Stop()

	// Клиент
	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	client := pb.NewNotesServiceClient(conn)

	// Подготавливаем контекст с userID (имитация работы AuthInterceptor)
	ctx := context.WithValue(context.Background(), "user_id", int64(42))

	// Вызов метода
	resp, err := client.CreateNote(ctx, &pb.CreateNoteRequest{
		Title:   "gRPC Title",
		Content: "gRPC Content",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(100), resp.Id)
	assert.Equal(t, "gRPC Title", resp.Title)
}

func TestGRPCGetNoteNotFound(t *testing.T) {
	mockSvc := &mockNoteServiceForGRPC{
		getNoteFunc: func(ctx context.Context, userID int64, id int64) (*model.Note, error) {
			return nil, service.ErrNoteNotFound
		},
	}

	server := grpc.NewServer()
	pb.RegisterNotesServiceServer(server, grpcimpl.NewNotesServer(mockSvc))
	listener := bufconn.Listen(1024 * 1024)
	go server.Serve(listener)
	defer server.Stop()

	conn, _ := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	defer conn.Close()
	client := pb.NewNotesServiceClient(conn)

	ctx := context.WithValue(context.Background(), "user_id", int64(42))
	_, err := client.GetNote(ctx, &pb.GetNoteRequest{Id: 999})
	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}
