package repository

import (
	"MainProject/internal/model"
	"context"
	"encoding/csv"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Repository — репозиторий для хранения сущностей разных типов
type Repository struct {
	notes    []model.Entity
	users    []model.Entity
	sessions []model.Entity
	tags     []model.Entity

	// Мьютексы для конкурентного доступа
	muNotes    sync.Mutex
	muUsers    sync.Mutex
	muSessions sync.Mutex
	muTags     sync.Mutex

	// Канал для приёма сущностей
	inputChan chan []model.Entity
	// Сохранение контекста для отслеживания завершения
	ctx context.Context
	// Указатель на WaitGroup для учёта горутины
	wg *sync.WaitGroup
}

// NewRepository создаёт новый репозиторий.
func NewRepository(ctx context.Context, wg *sync.WaitGroup) *Repository {
	// Создаём папку data/, если её нет
	err := os.MkdirAll("data", 0755) // 0755 — права: rwxr-xr-x
	if err != nil {
		log.Printf("Не удалось создать папку data/: %v", err)
		return nil
	}
	repo := &Repository{
		notes:     make([]model.Entity, 0),
		users:     make([]model.Entity, 0),
		sessions:  make([]model.Entity, 0),
		tags:      make([]model.Entity, 0),
		inputChan: make(chan []model.Entity, 100), // Буферизованный канал

		// Сохранение переданного контекста
		ctx: ctx,
		wg:  wg,
	}

	// Загрузка данных из файлов при старте
	repo.loadData()

	wg.Add(1)
	// Запуск горутины для обработки входящих сущностей
	go repo.processEntities()

	return repo
}

// Метод для получения канала для возможности сервисом отправлять данные
func (r *Repository) InputChannel() chan<- []model.Entity {
	return r.inputChan
}

// processEntities — горутина, непрерывно обрабатывающая входящие сущности из канала inputChan
func (r *Repository) processEntities() {
	// Откладываем вызов wg.Done() до завершения функции
	defer r.wg.Done()

	for {
		// Одновременное ожидание:
		// 1. Данных из канала inputChan
		// 2. Сигнала отмены из контекста (ctx.Done())
		select {
		// Бесконечный цикл чтения из канала inputChan
		case entities, ok := <-r.inputChan:
			// Если канал закрыт (ok == false), завершаем работу
			if !ok {
				return
			}
			log.Printf("Получено %d сущностей для обработки", len(entities))
			// Обработка каждой сущности в полученной группе
			for _, entity := range entities {
				// Определение конкретного типа сущности через type switch
				// Что позволяет корректно добавить объект в соответствующий слайс
				switch v := entity.(type) {
				case *model.Note:
					// 1. Блокировка доступа к слайсу notes на время модификации
					// 2. Добавление сущности в слайс
					// 3. Снятие блокировки
					r.muNotes.Lock()
					r.notes = append(r.notes, v)
					r.muNotes.Unlock()
				case *model.User:
					r.muUsers.Lock()
					r.users = append(r.users, v)
					r.muUsers.Unlock()
				case *model.Session:
					r.muSessions.Lock()
					r.sessions = append(r.sessions, v)
					r.muSessions.Unlock()
				case *model.Tag:
					r.muTags.Lock()
					r.tags = append(r.tags, v)
					r.muTags.Unlock()
				default:
					log.Printf("Неизвестный тип сущности: %T (значение: %v)", entity, entity)

				}
			}
		case <-r.ctx.Done():
			select {
			case <-r.inputChan: // Пытаемся прочитать, чтобы проверить, закрыт ли канал
				// Канал уже закрыт
			default:
				// Канал ещё открыт
				// Получен сигнал отмены (вызван cancel())
				// Закрытие канала inputChan
				close(r.inputChan)
				log.Println("Канал inputChan закрыт по сигналу контекста")
			}
			return // Завершение горутины
		}
	}
}

// Методы для сохранения данных в файлы
func (r *Repository) saveNotes() {
	r.muNotes.Lock()
	defer r.muNotes.Unlock()

	// Открываем файл для записи (создание нового или перезапись существующего)
	file, err := os.Create("data/notes.json")
	if err != nil {
		log.Printf("Ошибка при создании data/notes.json: %v", err)
		return
	}
	defer file.Close() // Закрытие файла

	// Создаём JSON-энкодер и записываем данные
	encoder := json.NewEncoder(file)
	err = encoder.Encode(r.notes)
	if err != nil {
		log.Printf("Ошибка при кодировании notes в JSON: %v", err)
	} else {
		log.Printf("Сохранено %d записей в data/notes.json", len(r.notes))
	}
}

func (r *Repository) saveUsers() {
	r.muUsers.Lock()
	defer r.muUsers.Unlock()

	file, err := os.Create("data/users.json")
	if err != nil {
		log.Printf("Ошибка при создании data/users.json: %v", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(r.users)
	if err != nil {
		log.Printf("Ошибка при кодировании users в JSON: %v", err)
	} else {
		log.Printf("Сохранено %d записей в data/users.json", len(r.notes))
	}
}

func (r *Repository) saveSessions() {
	r.muSessions.Lock()
	defer r.muSessions.Unlock()

	file, err := os.Create("data/sessions.json")
	if err != nil {
		log.Printf("Ошибка при создании data/sessions.json: %v", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(r.sessions)
	if err != nil {
		log.Printf("Ошибка при кодировании sessions в JSON: %v", err)
	} else {
		log.Printf("Сохранено %d записей в data/sessions.json", len(r.notes))
	}
}

func (r *Repository) saveTags() {
	r.muTags.Lock()
	defer r.muTags.Unlock()

	file, err := os.Create("data/tags.json")
	if err != nil {
		log.Printf("Ошибка при создании data/tags.json: %v", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(r.tags)
	if err != nil {
		log.Printf("Ошибка при кодировании tags в JSON: %v", err)
	} else {
		log.Printf("Сохранено %d записей в data/tags.json", len(r.notes))
	}
}

// Flush принудительно сохраняет все данные репозитория в файлы
// Вызыв перед завершением приложения для гарантии записи всех накопленных сущностей
func (r *Repository) Flush() {
	log.Println("Запущено принудительное сохранение всех данных...")

	r.saveNotes()
	r.saveUsers()
	r.saveSessions()
	r.saveTags()

	log.Println("Принудительное сохранение завершено.")
}

// Методы для загрузки данных из файлов
func (r *Repository) loadData() {
	r.loadNotes()
	r.loadUsers()
	r.loadSessions()
	r.loadTags()
}

// loadNotes загружает данные из notes.json в слайс notes
func (r *Repository) loadNotes() {
	file, err := os.Open("data/notes.json")
	if os.IsNotExist(err) {
		log.Printf("Файл notes.json не найден — пропускаем загрузку")
		return
	}
	if err != nil {
		println("Ошибка при открытии notes.json:", err.Error())
		return
	}
	defer file.Close()

	// Создаём JSON-декодер и читаем данные
	var rawNotes []model.Note
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&rawNotes)
	if err != nil {
		println("Ошибка при декодировании notes.json:", err.Error())
		return
	}

	// Преобразуем в []model.Entity
	var entities []model.Entity
	for _, note := range rawNotes {
		note.IsGenerated = false
		entities = append(entities, &note) // или &note, если нужно сохранить указатель
	}

	// Блокируем доступ к слайсу, записываем данные, разблокируем
	r.muNotes.Lock()
	r.notes = entities
	r.muNotes.Unlock()

}

func (r *Repository) loadUsers() {
	file, err := os.Open("data/users.json")
	if os.IsNotExist(err) {
		log.Printf("Файл users.json не найден — пропускаем загрузку")
		return
	}
	if err != nil {
		println("Ошибка при открытии users.json:", err.Error())
		return
	}
	defer file.Close()

	var rawUsers []model.User
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&rawUsers)
	if err != nil {
		println("Ошибка при декодировании users.json:", err.Error())
		return
	}

	var entities []model.Entity
	for _, user := range rawUsers {
		user.IsGenerated = false
		entities = append(entities, &user)
	}

	r.muUsers.Lock()
	r.users = entities
	r.muUsers.Unlock()

}

func (r *Repository) loadSessions() {
	file, err := os.Open("data/sessions.json")
	if os.IsNotExist(err) {
		log.Printf("Файл sessions.json не найден — пропускаем загрузку")
		return
	}
	if err != nil {
		println("Ошибка при открытии sessions.json:", err.Error())
		return
	}
	defer file.Close()

	var rawSessions []model.Session
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&rawSessions)
	if err != nil {
		println("Ошибка при декодировании sessions.json:", err.Error())
		return
	}

	var entities []model.Entity
	for _, session := range rawSessions {
		session.IsGenerated = false
		entities = append(entities, &session)
	}

	r.muSessions.Lock()
	r.sessions = entities
	r.muSessions.Unlock()
}

func (r *Repository) loadTags() {
	file, err := os.Open("data/tags.json")
	if os.IsNotExist(err) {
		log.Printf("Файл tags.json не найден — пропускаем загрузку")
		return
	}
	if err != nil {
		println("Ошибка при открытии tags.json:", err.Error())
		return
	}
	defer file.Close()

	var rawTags []model.Tag
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&rawTags)
	if err != nil {
		println("Ошибка при декодировании tags.json:", err.Error())
		return
	}

	var entities []model.Entity
	for _, tag := range rawTags {
		tag.IsGenerated = false
		entities = append(entities, &tag)
	}

	r.muTags.Lock()
	r.tags = entities
	r.muTags.Unlock()
}

// Методы для безопасного чтения (с мьютексами)
func (r *Repository) GetNotes() []model.Entity {
	r.muNotes.Lock()
	defer r.muNotes.Unlock()
	return r.notes
}

func (r *Repository) GetUsers() []model.Entity {
	r.muUsers.Lock()
	defer r.muUsers.Unlock()
	return r.users
}

func (r *Repository) GetSessions() []model.Entity {
	r.muSessions.Lock()
	defer r.muSessions.Unlock()
	return r.sessions
}

func (r *Repository) GetTags() []model.Entity {
	r.muTags.Lock()
	defer r.muTags.Unlock()
	return r.tags
}

// AddNote — добавляет заметку, генерирует ID
func (r *Repository) AddNote(note *model.Note) int64 {
	r.muNotes.Lock()
	id := int64(len(r.notes) + 1)
	note.SetID(id)
	r.notes = append(r.notes, note)
	r.muNotes.Unlock()
	return id
}

// FindNoteById — ищет заметку по ID
func (r *Repository) FindNoteById(id int64) (*model.Note, bool) {
	r.muNotes.Lock()
	defer r.muNotes.Unlock()
	for _, entity := range r.notes {
		if entity.(*model.Note).ID() == id {
			return entity.(*model.Note), true
		}
	}
	return nil, false
}

// UpdateNote — обновляет заметку
func (r *Repository) UpdateNote(id int64, updated *model.Note) bool {
	r.muNotes.Lock()
	defer r.muNotes.Unlock()
	for i, entity := range r.notes {
		if entity.(*model.Note).ID() == id {
			r.notes[i] = updated
			return true
		}
	}
	return false
}

// DeleteNote — удаляет заметку
func (r *Repository) DeleteNote(id int64) bool {
	r.muNotes.Lock()
	defer r.muNotes.Unlock()
	for i, entity := range r.notes {
		if entity.(*model.Note).ID() == id {
			r.notes = append(r.notes[:i], r.notes[i+1:]...)
			return true
		}
	}
	return false
}

// SaveNoteToCSV — сохраняет заметку в CSV
func (r *Repository) SaveNoteToCSV(note *model.Note) error {
	file, err := os.OpenFile("data/notes.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	record := []string{
		strconv.FormatInt(note.ID().(int64), 10),
		note.Title,
		note.Content,
		note.CreatedAt.Format(time.RFC3339),
		note.UpdatedAt.Format(time.RFC3339),
		strings.Join(note.Tags, ";"),
		strconv.FormatBool(note.IsPublic),
	}
	err = writer.Write(record)
	if err != nil {
		return err
	}
	writer.Flush()
	return nil
}
