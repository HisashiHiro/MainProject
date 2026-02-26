package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "MainProject/api/proto"
)

func main() {
	// Подключение к gRPC серверу
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Не удалось подключиться к серверу: %v", err)
	}
	defer conn.Close()

	client := pb.NewNotesServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Тестирование создания заметки
	log.Println("Тест 1: Создание заметки")
	createResp, err := client.CreateNote(ctx, &pb.CreateNoteRequest{
		Title:    "Тестовая заметка",
		Content:  "Это содержимое тестовой заметки",
		Tags:     []string{"тест", "gRPC"},
		IsPublic: true,
	})
	if err != nil {
		log.Fatalf("Ошибка при создании заметки: %v", err)
	}
	log.Printf("Создана заметка: ID=%d, Title=%s", createResp.Id, createResp.Title)

	// Тестирование получения заметки по ID
	log.Println("\nТест 2: Получение заметки по ID")
	getResp, err := client.GetNote(ctx, &pb.GetNoteRequest{Id: createResp.Id})
	if err != nil {
		log.Fatalf("Ошибка при получении заметки: %v", err)
	}
	log.Printf("Получена заметка: ID=%d, Title=%s, Content=%s",
		getResp.Id, getResp.Title, getResp.Content)

	// Тестирование получения всех заметок
	log.Println("\nТест 3: Получение всех заметок")
	listResp, err := client.ListNotes(ctx, &pb.ListNotesRequest{})
	if err != nil {
		log.Fatalf("Ошибка при получении списка заметок: %v", err)
	}
	log.Printf("Всего заметок: %d", listResp.TotalCount)
	for i, note := range listResp.Notes {
		log.Printf("  %d. ID=%d, Title=%s", i+1, note.Id, note.Title)
	}

	// Тестирование обновления заметки
	log.Println("\nТест 4: Обновление заметки")
	updateResp, err := client.UpdateNote(ctx, &pb.UpdateNoteRequest{
		Id:       createResp.Id,
		Title:    "Обновленная заметка",
		Content:  "Новое содержимое",
		Tags:     []string{"тест", "gRPC", "обновлено"},
		IsPublic: false,
	})
	if err != nil {
		log.Fatalf("Ошибка при обновлении заметки: %v", err)
	}
	log.Printf("Заметка обновлена: ID=%d, Title=%s, Public=%v",
		updateResp.Id, updateResp.Title, updateResp.IsPublic)

	// Тестирование удаления заметки
	log.Println("\nТест 5: Удаление заметки")
	deleteResp, err := client.DeleteNote(ctx, &pb.DeleteNoteRequest{Id: createResp.Id})
	if err != nil {
		log.Fatalf("Ошибка при удалении заметки: %v", err)
	}
	log.Printf("Результат удаления: Success=%v, Message=%s",
		deleteResp.Success, deleteResp.Message)

	log.Println("\nВсе тесты завершены успешно!")
}
