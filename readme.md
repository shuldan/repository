# `repository` — Простая реализация паттерна Repository для Go

[![Go CI](https://github.com/shuldan/repository/workflows/Go%20CI/badge.svg)](https://github.com/shuldan/repository/actions)
[![codecov](https://codecov.io/gh/shuldan/repository/branch/main/graph/badge.svg)](https://codecov.io/gh/shuldan/repository)
[![Go Report Card](https://goreportcard.com/badge/github.com/shuldan/repository)](https://goreportcard.com/report/github.com/shuldan/repository)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

Минималистичная и типобезопасная реализация паттерна Repository для Go. Работает с любыми типами данных без необходимости реализации интерфейсов.

---

## 🚀 Основные возможности

- **Простота**: работает с любыми структурами, не требует реализации интерфейсов
- **Типобезопасность**: полная поддержка Go Generics
- **Гибкость**: легко адаптируется под любую схему БД
- **Mapper Pattern**: явное разделение SQL-логики и бизнес-логики
- **Полный CRUD**: `Find`, `FindAll`, `FindBy`, `ExistsBy`, `CountBy`, `Save`, `Delete`
- **Динамические запросы**: поддержка произвольных SQL-условий

---

## 📦 Установка

```bash
go get github.com/shuldan/repository
```

Требования: Go 1.24+

### Установка инструментов разработки

```sh
make install-tools
```

Устанавливает:
- `golangci-lint` (v2.4.0)
- `goimports`
- `gosec`

---

## 🛠️ Быстрый старт

### 1. Определите вашу структуру данных

```go
package domain

type User struct {
    ID    string
    Name  string
    Email string
}
```

Всё! Не нужно реализовывать никакие интерфейсы.

### 2. Реализуйте Mapper

```go
package infrastructure

import (
    "context"
    "database/sql"
    "fmt"
    
    "github.com/shuldan/repository"
    "yourapp/domain"
)

type userMapper struct{}

func NewUserMapper() repository.Mapper[*domain.User] {
    return &userMapper{}
}

func (m *userMapper) Find(ctx context.Context, db *sql.DB, id string) *sql.Row {
    return db.QueryRowContext(ctx, "SELECT id, name, email FROM users WHERE id = ?", id)
}

func (m *userMapper) FindAll(ctx context.Context, db *sql.DB, limit, offset int) (*sql.Rows, error) {
    return db.QueryContext(ctx, "SELECT id, name, email FROM users LIMIT ? OFFSET ?", limit, offset)
}

func (m *userMapper) FindBy(ctx context.Context, db *sql.DB, conditions string, args []any) (*sql.Rows, error) {
    query := fmt.Sprintf("SELECT id, name, email FROM users WHERE %s", conditions)
    return db.QueryContext(ctx, query, args...)
}

func (m *userMapper) ExistsBy(ctx context.Context, db *sql.DB, conditions string, args []any) (bool, error) {
    query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM users WHERE %s)", conditions)
    var exists bool
    err := db.QueryRowContext(ctx, query, args...).Scan(&exists)
    return exists, err
}

func (m *userMapper) CountBy(ctx context.Context, db *sql.DB, conditions string, args []any) (int64, error) {
    query := fmt.Sprintf("SELECT COUNT(*) FROM users WHERE %s", conditions)
    var count int64
    err := db.QueryRowContext(ctx, query, args...).Scan(&count)
    return count, err
}

func (m *userMapper) Save(ctx context.Context, db *sql.DB, user *domain.User) error {
    query := `
        INSERT INTO users (id, name, email) VALUES (?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET name = ?, email = ?
    `
    _, err := db.ExecContext(ctx, query, user.ID, user.Name, user.Email, user.Name, user.Email)
    return err
}

func (m *userMapper) Delete(ctx context.Context, db *sql.DB, id string) error {
    _, err := db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
    return err
}

func (m *userMapper) FromRow(row *sql.Row) (*domain.User, error) {
    var user domain.User
    err := row.Scan(&user.ID, &user.Name, &user.Email)
    if err != nil {
        return nil, err
    }
    return &user, nil
}

func (m *userMapper) FromRows(rows *sql.Rows) ([]*domain.User, error) {
    var users []*domain.User
    defer rows.Close()
    
    for rows.Next() {
        var user domain.User
        if err := rows.Scan(&user.ID, &user.Name, &user.Email); err != nil {
            return nil, err
        }
        users = append(users, &user)
    }
    
    return users, rows.Err()
}
```

### 3. Используйте репозиторий

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"
    
    _ "github.com/mattn/go-sqlite3"
    "github.com/shuldan/repository"
    
    "yourapp/domain"
    "yourapp/infrastructure"
)

func main() {
    // Открываем БД
    db, err := sql.Open("sqlite3", "app.db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // Создаём таблицу
    db.Exec(`
        CREATE TABLE IF NOT EXISTS users (
            id TEXT PRIMARY KEY,
            name TEXT NOT NULL,
            email TEXT NOT NULL
        )
    `)
    
    // Создаём репозиторий
    mapper := infrastructure.NewUserMapper()
    repo := repository.NewRepository[*domain.User](db, mapper)
    
    ctx := context.Background()
    
    // Сохраняем пользователя
    user := &domain.User{
        ID:    "user-1",
        Name:  "Alice",
        Email: "alice@example.com",
    }
    
    if err := repo.Save(ctx, user); err != nil {
        log.Fatal(err)
    }
    fmt.Println("✅ User saved")
    
    // Находим по ID
    found, err := repo.Find(ctx, "user-1")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("📦 Found: %s (%s)\n", found.Name, found.Email)
    
    // Ищем по условию
    users, err := repo.FindBy(ctx, "email LIKE ?", []any{"%example.com"})
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("🔍 Found %d users\n", len(users))
    
    // Проверяем существование
    exists, err := repo.ExistsBy(ctx, "name = ?", []any{"Alice"})
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("✓ Exists: %v\n", exists)
    
    // Считаем записи
    count, err := repo.CountBy(ctx, "1=1", nil)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("📊 Total users: %d\n", count)
    
    // Получаем все с пагинацией
    all, err := repo.FindAll(ctx, 10, 0) // limit=10, offset=0
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("📄 Page: %d users\n", len(all))
    
    // Удаляем
    if err := repo.Delete(ctx, "user-1"); err != nil {
        log.Fatal(err)
    }
    fmt.Println("🗑️ User deleted")
}
```

---

## 🧱 Архитектура

### `Mapper[T]`

Интерфейс для преобразования между SQL и вашими структурами:

```go
type Mapper[T any] interface {
    // SQL-запросы
    Find(ctx context.Context, db *sql.DB, id string) *sql.Row
    FindAll(ctx context.Context, db *sql.DB, limit, offset int) (*sql.Rows, error)
    FindBy(ctx context.Context, db *sql.DB, conditions string, args []any) (*sql.Rows, error)
    ExistsBy(ctx context.Context, db *sql.DB, conditions string, args []any) (bool, error)
    CountBy(ctx context.Context, db *sql.DB, conditions string, args []any) (int64, error)
    Save(ctx context.Context, db *sql.DB, aggregate T) error
    Delete(ctx context.Context, db *sql.DB, id string) error
    
    // Преобразования
    FromRow(row *sql.Row) (T, error)
    FromRows(rows *sql.Rows) ([]T, error)
}
```

### `Repository[T]`

Унифицированный интерфейс доступа к данным:

```go
type Repository[T any] interface {
    Find(ctx context.Context, id string) (T, error)
    FindAll(ctx context.Context, limit, offset int) ([]T, error)
    FindBy(ctx context.Context, conditions string, args []any) ([]T, error)
    ExistsBy(ctx context.Context, conditions string, args []any) (bool, error)
    CountBy(ctx context.Context, conditions string, args []any) (int64, error)
    Save(ctx context.Context, aggregate T) error
    Delete(ctx context.Context, id string) error
}
```

---

## 🎯 Примеры использования

### Поиск с фильтрацией

```go
// Активные пользователи
activeUsers, err := repo.FindBy(ctx, "status = ?", []any{"active"})

// Пользователи с определённым доменом email
gmailUsers, err := repo.FindBy(ctx, "email LIKE ?", []any{"%@gmail.com"})

// Сложные условия
premiumUsers, err := repo.FindBy(ctx, 
    "status = ? AND created_at > ? AND role IN (?, ?)",
    []any{"active", time.Now().AddDate(0, -1, 0), "premium", "vip"},
)
```

### Проверка и подсчёт

```go
// Проверить, есть ли пользователь с таким email
exists, err := repo.ExistsBy(ctx, "email = ?", []any{"test@example.com"})
if exists {
    return errors.New("email already taken")
}

// Подсчитать количество активных пользователей
activeCount, err := repo.CountBy(ctx, "status = ?", []any{"active"})
fmt.Printf("Active users: %d\n", activeCount)
```

### Пагинация

```go
page := 1
pageSize := 20
offset := (page - 1) * pageSize

users, err := repo.FindAll(ctx, pageSize, offset)

// Общее количество для пагинации
total, err := repo.CountBy(ctx, "1=1", nil)
totalPages := (total + int64(pageSize) - 1) / int64(pageSize)
```

### Обработка ошибок

```go
user, err := repo.Find(ctx, "user-123")
if err != nil {
    if errors.Is(err, repository.ErrEntityNotFound) {
        return fmt.Errorf("user not found")
    }
    return fmt.Errorf("database error: %w", err)
}
```

---

## 🔧 Работа с проектом

### Локальная проверка

```sh
make all
```

Выполняет форматирование, линтинг, security-сканирование и тесты.

### CI проверка

```sh
make ci
```

### Только тесты

```sh
make test              # Быстрые тесты
make test-coverage     # С отчётом о покрытии
```

### Форматирование

```sh
make fmt               # Автоформат
make fmt-check         # Проверка без изменений
```

---

## ✨ Преимущества

### Минимализм
Не нужно реализовывать интерфейсы или следовать строгим соглашениям. Работает с любыми структурами.

### Чистая архитектура
- **Domain** (`User`) — просто структура данных
- **Mapper** — вся SQL-логика изолирована
- **Repository** — единообразный интерфейс доступа

### Типобезопасность
Компилятор гарантирует корректность типов:

```go
repo := repository.NewRepository[*User](db, mapper)
user, err := repo.Find(ctx, "123")  // user имеет тип *User
```

### Тестируемость
Легко создавать моки для тестирования:

```go
type mockMapper struct{}

func (m *mockMapper) Find(ctx, db, id) *sql.Row { /* ... */ }
// ... остальные методы

repo := repository.NewRepository[*User](db, &mockMapper{})
```

### Переносимость
Меняйте хранилище без изменения бизнес-логики:
- SQL → NoSQL
- Postgres → MySQL
- Database → In-Memory Cache

---

## 📊 Производительность

- Минимальные аллокации благодаря дженерикам
- Прямое преобразование `sql.Rows` → структуры
- Поддержка batch-операций через `FindAll` и `FindBy`

---

## 🧪 Тестирование

Пакет покрыт тестами > 70%. Запустите тесты:

```sh
make test-coverage
```

Откройте отчёт:

```sh
open coverage.html
```

---

## 📄 Лицензия

Проект распространяется под лицензией [MIT](LICENSE).

---

## 🤝 Вклад в проект

Приветствуются PR и issues! 

Перед отправкой PR:

```sh
make all  # Проверяет форматирование, линтинг и тесты
```

Требования:
- Покрытие тестами новой функциональности
- Соответствие `golangci-lint`
- Документация в коде

---

## 📚 Полезные ссылки

- [Repository Pattern](https://martinfowler.com/eaaCatalog/repository.html) — Martin Fowler
- [Data Mapper Pattern](https://martinfowler.com/eaaCatalog/dataMapper.html)
- [Go Generics](https://go.dev/doc/tutorial/generics)

---

## 💡 FAQ

**Q: Можно ли использовать со сложными JOIN-запросами?**  
A: Да, реализуйте собственные методы в Mapper для специфичных запросов.

**Q: Поддерживаются ли транзакции?**  
A: Передайте `*sql.Tx` вместо `*sql.DB` в методы Mapper.

**Q: Как работать с UUID вместо string ID?**  
A: Измените сигнатуру методов `Find`/`Delete` в вашем Mapper:
```go
func (m *mapper) Find(ctx, db, id uuid.UUID) *sql.Row { ... }
```

**Q: Нужно ли использовать указатели `*User`?**  
A: Зависит от вас. Можно использовать `User` или `*User` — оба варианта работают.

---

> **Автор**: MSeytumerov  
> **Репозиторий**: [github.com/shuldan/repository](https://github.com/shuldan/repository)  
> **Go Version**: 1.24.2
