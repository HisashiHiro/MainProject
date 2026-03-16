# Веб-приложение "Заметки"

## 📝 Описание
Это многофункциональное веб-приложение для управления личными заметками, разработанное в рамках проектной работы в OTUS.  
Приложение позволяет пользователям безопасно создавать, просматривать, редактировать и удалять заметки, добавлять к ним теги и устанавливать срок давности.  
Поддерживается **аутентификация и авторизация** (сессии для веб-интерфейса, JWT для API), благодаря чему каждый пользователь видит только свои заметки.

Проект построен на **Go** и включает:
- **Веб-интерфейс** на HTML-шаблонах (Gin)
- **JSON API** (HTTP + Swagger)
- **gRPC API** (с кодом из proto)
- Два варианта хранилища: **PostgreSQL** или **MongoDB** (выбирается через переменную окружения)
- **Redis** для аудита изменений заметок
- **Docker Compose** для развёртывания всех зависимостей и самого приложения
- **CI/CD** (GitHub Actions) с линтером, юнит-тестами и сборкой

## ✨ Функциональность
### Основные возможности
- ✅ Регистрация и вход пользователей (веб и API)
- ✅ CRUD для заметок (заголовок, содержимое, теги, публичность, срок давности)
- ✅ Привязка заметок к владельцу (изоляция данных)
- ✅ Поддержка тегов (Many to many)
- ✅ Срок давности заметки
- ✅ Аудит изменений в Redis (события create/update/delete с TTL)

### Интерфейсы
- 🌐 **Веб-интерфейс** (`localhost:8080`) — дружелюбные страницы на HTML
- 📡 **REST API** (`/api`) — JSON, документирован через Swagger (`/swagger/index.html`)
- 🔌 **gRPC API** (`localhost:50051`) — высокопроизводительный RPC

## 🛠 Технологии
| Компонент       | Технологии                                                                 |
|-----------------|----------------------------------------------------------------------------|
| Backend         | Go 1.25.1, Gin, sqlx, MongoDB Go Driver, gRPC, goose (миграции)            |
| Аутентификация  | JWT (API), cookie-сессии (веб)                                             |
| Базы данных     | PostgreSQL 15 / MongoDB 7, Redis 7 для аудита                              |
| Контейнеризация | Docker, Docker Compose                                                     |
| CI/CD           | GitHub Actions (линтер, тесты с -race, сборка бинарника)                   |
| Документация    | Swagger (swaggo), Protobuf                                                 |

## 🏗 Архитектура
Проект организован по принципу **чистой архитектуры**:

MainProject/  
├── .github/  
│   └── workflows/  
│       ├── main.yml  
├── api/  
│   └── proto/  
│       ├── notes_grpc.pb.go  
│       ├── notes.pb.go  
│       └── notes.proto  
├── cmd/  
│   ├── grpc_client/  
│   │   └── main.go  
│   ├── NotesApp/  
│   │   ├── data/  
│   │   │   ├── notes.csv  
│   │   │   ├── notes.json  
│   │   │   ├── sessions.json  
│   │   │   ├── tags.json  
│   │   │   └── users.json  
│   │   └──  docs/  
│   │      ├── docs.go  
│   │      ├── swagger.json  
│   │      └── swagger.yaml  
│   └── main.go  
├── data/  
├── internal/  
│   ├── grpc/  
│   │   ├── auth_interceptor.go  
│   │   ├── server_test.go  
│   │   └── server.go  
│   ├── handlers/  
│   │   ├── handlers_web.go  
│   │   ├── handlers.go  
│   │   └── handlers_test.go  
│   ├── model/  
│   │   └── model.go  
│   ├── repository/  
│   │   ├── audit.go  
│   │   ├── mongo_test.go  
│   │   ├── mongo.go  
│   │   ├── postgres_test.go  
│   │   └── postgres.go  
│   └── service/  
│       ├── notes_test.go  
│       ├── notes.go  
│       └── user.go  
├── migrations/  
│   ├── _create_notes.sql  
│   ├── _create_users.sql  
│   ├── _create_tags.sql  
│   ├── _create_sessions.sql  
│   ├── _create_notes_tags.sql  
│   ├── _add_user_id_to_notes.sql  
│   └── _add_expires_at_to_notes.sql  
├── templates/  
│   ├── create_note.html  
│   ├── edit_note.html  
│   ├── error.html  
│   ├── index.html  
│   ├── login.html  
│   ├── register.html  
│   └── view_note.html  
├── .env  
├── .gitignore  
├── docker-compose.yml  
├── Dockerfile  
├── generate.bat  
├── go.mod  
├── go.sum  
├── golangci.yml  
└── README.md  

## 🚀 Запуск проекта

### Предварительные требования
- Установленные **Docker** и **Docker Compose** (или Go 1.25.1 для локального запуска)
- (Опционально) `protoc` для генерации gRPC-кода (см. `generate.bat`)

### Переменные окружения
Создайте файл `.env` в корне проекта (пример см. в `.env.example`). Основные параметры:
```env
JWT_SECRET=your-secret
SESSION_SECRET=your-session-secret
LOGIN=admin               # тестовый пользователь
PASSWORD=secret

# Выбор БД: postgres или mongo
DB_TYPE=postgres

# PostgreSQL
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres-password-2026
POSTGRES_DB=notes_app
DATABASE_URL=postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable

# MongoDB
MONGO_URI=mongodb://mongo:27017
MONGO_DB=notes_app

# Redis
REDIS_ADDR=redis:6379
REDIS_PASSWORD=redis-password-2026
REDIS_AUDIT_TTL=168h
```


**Запуск через Docker Compose (рекомендуется)**
```
docker-compose up -d
```

Будут запущены контейнеры:

- postgres (или mongo — в зависимости от выбранного образа в compose)

- redis

- app (само приложение)

После запуска:

- Веб-интерфейс: http://localhost:8080

- Swagger: http://localhost:8080/swagger/index.html

- gRPC-сервер: localhost:50051

**Локальный запуск (без Docker)**

Установите зависимости:
```
go mod download
```
Запустите приложение:
```
go run ./cmd/NotesApp/main.go
```
(Убедитесь, что PostgreSQL/MongoDB и Redis запущены локально и переменные окружения настроены.)

## 🧪 Тестирование

Юнит-тесты (core-логика)
```
go test -v -race -count=1 ./internal/service/...
```
Интеграционные тесты (PostgreSQL)
```
go test -v -tags=integration ./internal/repository/...
```
(Для MongoDB аналогично, но требуется запущенный контейнер MongoDB и установленный тег сборки, если используется //go:build integration.)

**Тесты транспортного уровня (HTTP, gRPC)**  
Примеры тестов находятся в internal/handlers/handlers_test.go и internal/grpc/server_test.go (используют моки сервисов).

## 🔄 CI/CD

Проект использует GitHub Actions (файл .github/workflows/main.yml). Пайплайн включает:

- Установку Go
- Кэширование модулей
- Установку golangci-lint (последняя версия)
- Запуск линтера с конфигом .golangci.yml
- Запуск юнит-тестов с флагами -race -count 100
- Сборку бинарного файла сервиса
- Сохранение артефакта сборки
- Пайплайн активируется при push в ветку master и при создании pull request.

## 📚 Документация API

### Swagger (OpenAPI)
После запуска приложения перейдите по адресу:

text
http://localhost:8080/swagger/index.html
Там описаны все эндпоинты REST API, модели и схема авторизации (Bearer JWT).

### gRPC
Описание сервиса и сообщений находится в api/proto/notes.proto. Пример клиента — в cmd/grpc_client/.
Для генерации кода из protobuf используйте:

```
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    api/proto/notes.proto
```
(или запустите generate.bat на Windows.)

### 📦 Структура базы данных (PostgreSQL)

Основные таблицы:

- users — пользователи (поля: id, username, email, password_hash, last_login, …)
- notes — заметки (id, user_id, title, content, expires_at, is_public, …)
- tags — теги (id, name)
- note_tags — связь многие-ко-многим
- sessions — сессии для веб-аутентификации

Для MongoDB — аналогичные коллекции.