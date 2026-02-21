# `repository` — Типобезопасный Generic Repository для Go

[![Go CI](https://github.com/shuldan/repository/workflows/Go%20CI/badge.svg)](https://github.com/shuldan/repository/actions)
[![codecov](https://codecov.io/gh/shuldan/repository/branch/main/graph/badge.svg)](https://codecov.io/gh/shuldan/repository)
[![Go Report Card](https://goreportcard.com/badge/github.com/shuldan/repository)](https://goreportcard.com/report/github.com/shuldan/repository)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

Декларативная типобезопасная реализация паттерна Repository для Go с поддержкой Generics. Нулевые внешние зависимости — работает только со стандартной библиотекой `database/sql`.

---

## Основные возможности

- **Типобезопасность** — полная поддержка Go Generics, никаких `interface{}`
- **Декларативный маппинг** — описываете таблицу и функции сканирования, SQL генерируется автоматически
- **Простые и составные агрегаты** — `Simple` для одной таблицы, `Composite` для агрегата с дочерними сущностями
- **Составной первичный ключ** — поддержка одиночных и составных PK (`(account_id, role_id)`)
- **Спецификации** — типобезопасный построитель WHERE-условий (`Eq`, `In`, `Like`, `And`, `Or`, `Not`, `Raw` и др.)
- **Fluent Query API** — цепочечный построитель запросов с `Where`, `OrderBy`, `Limit`, `Offset`
- **Keyset-пагинация** — курсорная пагинация через `Page` / `After` / `Before`
- **Мультидиалектность** — PostgreSQL, MySQL, SQLite из коробки
- **Soft Delete** — встроенная поддержка мягкого удаления
- **Optimistic Locking** — контроль конкурентных изменений через колонку версии
- **Транзакции** — `SaveTx` / `DeleteTx` для работы внутри транзакций
- **Автоматический Upsert** — `INSERT ... ON CONFLICT` / `ON DUPLICATE KEY UPDATE`
- **Нулевые внешние зависимости** — только `database/sql`

---

## Установка

```bash
go get github.com/shuldan/repository
```

Требования: Go 1.24+

---

## Быстрый старт

### 1. Определите доменную модель

Пакет спроектирован для работы с инкапсулированными агрегатами. Снимок (Snapshot) служит промежуточным представлением для персистентности:

```go
// domain/user.go
package domain

type User struct {
    id    string
    name  string
    email string
}

func NewUser(id, name, email string) *User {
    return &User{id: id, name: name, email: email}
}

func (u *User) ID() string    { return u.id }
func (u *User) Name() string  { return u.name }
func (u *User) Email() string { return u.email }

// UserSnapshot — плоское представление для персистентности
type UserSnapshot struct {
    ID    string
    Name  string
    Email string
}

// Snapshot возвращает плоский снимок агрегата
func (u *User) Snapshot() UserSnapshot {
    return UserSnapshot{ID: u.id, Name: u.name, Email: u.email}
}

// Restore восстанавливает агрегат из снимка
func (s UserSnapshot) Restore() *User {
    return &User{id: s.ID, name: s.Name, email: s.Email}
}
```

### 2. Опишите маппинг

```go
// infrastructure/user_repository.go
package infrastructure

import (
    "database/sql"

    "github.com/shuldan/repository"
    "yourapp/domain"
)

var userTable = repository.Table{
    Name:       "users",
    PrimaryKey: []string{"id"},
    Columns:    []string{"id", "name", "email"},
}

func scanUser(sc repository.Scanner) (*domain.User, error) {
    var s domain.UserSnapshot
    if err := sc.Scan(&s.ID, &s.Name, &s.Email); err != nil {
        return nil, err
    }
    return s.Restore(), nil
}

func userValues(u *domain.User) []any {
    s := u.Snapshot()
    return []any{s.ID, s.Name, s.Email}
}

func NewUserRepository(db *sql.DB) *repository.Repository[*domain.User] {
    return repository.New(db, repository.Postgres(), repository.Simple(
        repository.SimpleConfig[*domain.User]{
            Table:  userTable,
            Scan:   scanUser,
            Values: userValues,
        },
    ))
}
```

### 3. Используйте репозиторий

```go
package main

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "log"

    _ "github.com/lib/pq"

    "github.com/shuldan/repository"
    "yourapp/domain"
    "yourapp/infrastructure"
)

func main() {
    db, err := sql.Open("postgres", "postgres://localhost/mydb?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    repo := infrastructure.NewUserRepository(db)
    ctx := context.Background()

    // Сохранение (Upsert)
    user := domain.NewUser("u-1", "Alice", "alice@example.com")
    if err := repo.Save(ctx, user); err != nil {
        log.Fatal(err)
    }

    // Поиск по ID
    found, err := repo.Find(ctx, "u-1")
    if err != nil {
        if errors.Is(err, repository.ErrNotFound) {
            fmt.Println("пользователь не найден")
            return
        }
        log.Fatal(err)
    }
    fmt.Printf("Найден: %s (%s)\n", found.Name(), found.Email())

    // Удаление
    if err := repo.Delete(ctx, "u-1"); err != nil {
        log.Fatal(err)
    }
}
```

---

## Содержание

- [Диалекты](#диалекты)
- [Описание таблицы](#описание-таблицы)
- [Составной первичный ключ](#составной-первичный-ключ)
- [Маппинг: Simple и Composite](#маппинг-simple-и-composite)
- [CRUD-операции](#crud-операции)
- [Спецификации (Spec)](#спецификации-spec)
- [Fluent Query API](#fluent-query-api)
- [Keyset-пагинация](#keyset-пагинация)
- [Soft Delete](#soft-delete)
- [Optimistic Locking](#optimistic-locking)
- [Транзакции](#транзакции)
- [Составные агрегаты (Composite)](#составные-агрегаты-composite)
- [Ошибки](#ошибки)
- [Полный пример](#полный-пример)
- [Разработка](#разработка)

---

## Диалекты

Пакет поддерживает три SQL-диалекта. Диалект определяет формат плейсхолдеров, синтаксис Upsert, функцию текущего времени и оператор нечувствительного к регистру поиска:

```go
repository.Postgres() // $1, $2, ... | ON CONFLICT ... DO UPDATE | NOW()     | ILIKE
repository.MySQL()    // ?, ?, ...   | ON DUPLICATE KEY UPDATE   | NOW()     | LIKE
repository.SQLite()   // ?, ?, ...   | ON CONFLICT ... DO UPDATE | datetime('now') | LIKE
```

Диалект передаётся при создании репозитория:

```go
repo := repository.New(db, repository.Postgres(), mapping)
```

---

## Описание таблицы

Структура `Table` определяет метаданные таблицы:

```go
repository.Table{
    Name:       "users",           // имя таблицы
    PrimaryKey: []string{"id"},    // первичный ключ (одиночный или составной)
    Columns:    []string{          // все колонки (порядок важен — совпадает с Scan и Values)
        "id", "name", "email", "version", "created_at", "updated_at",
    },

    // Опциональные поля:
    VersionColumn: "version",      // Optimistic Locking
    SoftDelete:    "deleted_at",   // Soft Delete
    CreatedAt:     "created_at",   // автоматически подставляется NOW() при INSERT
    UpdatedAt:     "updated_at",   // автоматически подставляется NOW() при INSERT и UPDATE
}
```

Поля `CreatedAt` и `UpdatedAt` **не передаются** в `Values` — они заполняются на уровне SQL автоматически. Колонка `VersionColumn` **передаётся** в `Values`, а при UPDATE автоматически инкрементируется на уровне SQL.

---

## Составной первичный ключ

Для таблиц-связок и других случаев, когда первичный ключ состоит из нескольких колонок:

```go
var accountRoleTable = repository.Table{
    Name:       "account_roles",
    PrimaryKey: []string{"account_id", "role_id"}, // составной ключ
    Columns:    []string{"account_id", "role_id"},
    CreatedAt:  "assigned_at",
}
```

### Как это работает

| Операция | Генерируемый SQL (PostgreSQL) |
|----------|-------------------------------|
| `Save` | `INSERT INTO account_roles (...) VALUES (...) ON CONFLICT (account_id, role_id) DO NOTHING` |
| `Delete` | `DELETE FROM account_roles WHERE account_id = $1 AND role_id = $2` |
| `Find` | `SELECT ... FROM account_roles WHERE (account_id = $1) AND (role_id = $2)` |

Если все колонки таблицы являются частью первичного ключа (нечего обновлять), генерируется `DO NOTHING` (PostgreSQL/SQLite) или `INSERT IGNORE` (MySQL).

### Использование `Find` и `Delete` с составным ключом

Методы `Find` и `Delete` принимают variadic-аргументы — количество должно совпадать с количеством колонок в `PrimaryKey`:

```go
// Одиночный PK
user, err := repo.Find(ctx, "u-1")
err = repo.Delete(ctx, "u-1")

// Составной PK — аргументы в порядке объявления PrimaryKey
role, err := repo.Find(ctx, accountID, roleID)
err = repo.Delete(ctx, accountID, roleID)
```

При несовпадении количества аргументов возвращается ошибка.

### Полный пример: связующая таблица

```go
var accountRoleTable = repository.Table{
    Name:       "account_roles",
    PrimaryKey: []string{"account_id", "role_id"},
    Columns:    []string{"account_id", "role_id"},
    CreatedAt:  "assigned_at",
}

func NewAccountRoleRepository(db *sql.DB) *repository.Repository[*AccountRole] {
    return repository.New(db, repository.Postgres(), repository.Simple(
        repository.SimpleConfig[*AccountRole]{
            Table: accountRoleTable,
            Scan: func(sc repository.Scanner) (*AccountRole, error) {
                var s AccountRoleSnapshot
                if err := sc.Scan(&s.AccountID, &s.RoleID); err != nil {
                    return nil, err
                }
                return s.Restore(), nil
            },
            Values: func(ar *AccountRole) []any {
                s := ar.Snapshot()
                return []any{s.AccountID, s.RoleID}
            },
        },
    ))
}

// Использование
repo := NewAccountRoleRepository(db)
err := repo.Save(ctx, role)                            // INSERT ... ON CONFLICT DO NOTHING
err = repo.Delete(ctx, accountID, roleID)              // DELETE WHERE account_id = $1 AND role_id = $2
roles, err := repo.FindBy(ctx, repository.Eq("account_id", accountID))  // SELECT WHERE account_id = $1
```

### Keyset-пагинация с составным PK

При использовании `Page` все колонки первичного ключа автоматически добавляются в `ORDER BY` для гарантии детерминированного порядка:

```go
// PrimaryKey: []string{"account_id", "role_id"}
// ORDER BY автоматически получит: ORDER BY account_id ASC, role_id ASC
page, err := repo.Query(ctx).PageSize(20).Page(extractor)
```

---

## Маппинг: Simple и Composite

### Simple — одна таблица

Для сущностей, хранящихся в одной таблице:

```go
mapping := repository.Simple(repository.SimpleConfig[*User]{
    Table:  userTable,
    Scan:   func(sc repository.Scanner) (*User, error) { /* ... */ },
    Values: func(u *User) []any { /* ... */ },
})
```

| Функция | Описание |
|---------|----------|
| `Scan`   | Читает строку из `Scanner` (совместим с `*sql.Row` и `*sql.Rows`) и возвращает агрегат |
| `Values` | Возвращает срез значений колонок в порядке `Table.Columns` для Upsert |

### Composite — агрегат с дочерними таблицами

Для агрегатов, включающих связанные сущности (например, Заказ + Позиции):

```go
mapping := repository.Composite(repository.CompositeConfig[*Order, *OrderSnapshot]{
    Table:     orderTable,
    Relations: []repository.Relation{itemsRelation},
    ScanRoot:  func(sc repository.Scanner) (*OrderSnapshot, error) { /* ... */ },
    ScanChild: func(table string, sc repository.Scanner, snap *OrderSnapshot) error { /* ... */ },
    Build:     func(snap *OrderSnapshot) (*Order, error) { /* ... */ },
    Decompose: func(o *Order) repository.CompositeValues { /* ... */ },
    ExtractPK: func(snap *OrderSnapshot) string { /* ... */ },
})
```

| Функция     | Описание |
|-------------|----------|
| `ScanRoot`  | Сканирует строку корневой таблицы в промежуточный снимок |
| `ScanChild` | Сканирует строку дочерней таблицы и добавляет данные в снимок |
| `Build`     | Собирает финальный агрегат из заполненного снимка |
| `Decompose` | Разбирает агрегат на значения корневой строки и дочерних строк |
| `ExtractPK` | Извлекает первичный ключ из снимка для загрузки связей |

Подробнее в разделе [Составные агрегаты (Composite)](#составные-агрегаты-composite).

---

## CRUD-операции

Все методы принимают `context.Context` первым аргументом:

```go
repo := repository.New(db, repository.Postgres(), mapping)
```

### Find — поиск по ID

```go
// Одиночный PK
user, err := repo.Find(ctx, "u-1")
if errors.Is(err, repository.ErrNotFound) {
    // не найден
}

// Составной PK
role, err := repo.Find(ctx, accountID, roleID)
```

### FindBy — поиск по спецификации

```go
users, err := repo.FindBy(ctx, repository.Eq("status", "active"))
```

Передайте `nil` для выборки всех записей:

```go
all, err := repo.FindBy(ctx, nil)
```

### ExistsBy — проверка существования

```go
exists, err := repo.ExistsBy(ctx, repository.Eq("email", "alice@example.com"))
```

### CountBy — подсчёт

```go
count, err := repo.CountBy(ctx, repository.Eq("status", "active"))
```

### Save — создание или обновление (Upsert)

```go
err := repo.Save(ctx, user)
```

Генерирует `INSERT ... ON CONFLICT DO UPDATE` (PostgreSQL/SQLite) или `INSERT ... ON DUPLICATE KEY UPDATE` (MySQL). Если все колонки являются частью первичного ключа, генерируется `DO NOTHING` / `INSERT IGNORE`.

### Delete — удаление по ID

```go
// Одиночный PK
err := repo.Delete(ctx, "u-1")

// Составной PK
err := repo.Delete(ctx, accountID, roleID)
```

При включённом Soft Delete выполняет `UPDATE ... SET deleted_at = NOW()`.

---

## Спецификации (Spec)

Спецификации — типобезопасный способ построения WHERE-условий. Каждая спецификация реализует интерфейс:

```go
type Spec interface {
    ToSQL(d Dialect, offset int) (sql string, args []any, nextOffset int)
}
```

### Операторы сравнения

```go
repository.Eq("status", "active")     // status = $1
repository.NotEq("role", "guest")     // role != $1
repository.Gt("age", 18)              // age > $1
repository.Gte("age", 18)             // age >= $1
repository.Lt("price", 100)           // price < $1
repository.Lte("price", 100)          // price <= $1
```

### IN / NOT IN

```go
repository.In("status", "active", "pending")      // status IN ($1, $2)
repository.NotIn("role", "banned", "deleted")      // role NOT IN ($1, $2)
```

При пустом списке значений: `In` → `FALSE`, `NotIn` → `TRUE`.

### LIKE / ILIKE

```go
repository.Like("name", "%alice%")     // name LIKE $1
repository.ILike("name", "%alice%")    // name ILIKE $1  (Postgres)
                                       // name LIKE $1   (MySQL/SQLite)
```

### BETWEEN

```go
repository.Between("age", 18, 65)      // age BETWEEN $1 AND $2
```

### NULL-проверки

```go
repository.IsNull("deleted_at")        // deleted_at IS NULL
repository.IsNotNull("deleted_at")     // deleted_at IS NOT NULL
```

### Логические комбинации

```go
repository.And(
    repository.Eq("status", "active"),
    repository.Gte("age", 18),
)
// (status = $1) AND (age >= $2)

repository.Or(
    repository.Eq("role", "admin"),
    repository.Eq("role", "moderator"),
)
// (role = $1) OR (role = $2)

repository.Not(repository.Eq("status", "banned"))
// NOT (status = $1)
```

Пустой `And` возвращает `TRUE`, пустой `Or` возвращает `FALSE`. Одиночный элемент разворачивается без скобок.

### Raw SQL

Для нестандартных условий:

```go
repository.Raw("age > $1 AND score < $2", 18, 100)
```

Плейсхолдеры `$1`, `$2` автоматически заменяются на формат текущего диалекта.

### Комплексный пример

```go
spec := repository.And(
    repository.Eq("status", "active"),
    repository.Or(
        repository.ILike("name", "%alice%"),
        repository.ILike("email", "%alice%"),
    ),
    repository.Between("created_at", startDate, endDate),
    repository.IsNull("deleted_at"),
)
users, err := repo.FindBy(ctx, spec)
```

---

## Fluent Query API

Метод `Query` возвращает построитель запросов с цепочечным API:

```go
q := repo.Query(ctx)
```

### Where — добавление условий

```go
q.Where(repository.Eq("status", "active"))
q.Where(repository.Gte("age", 18))
// Несколько Where объединяются через AND
```

### OrderBy — сортировка

```go
q.OrderBy("created_at", repository.Desc)
q.OrderBy("name", repository.Asc)
```

### Limit / Offset

```go
q.Limit(10).Offset(20)
```

### All — получить все результаты

```go
users, err := repo.Query(ctx).
    Where(repository.Eq("status", "active")).
    OrderBy("name", repository.Asc).
    Limit(50).
    All()
```

### First — получить первый результат

Автоматически устанавливает `LIMIT 1`. Возвращает `ErrNotFound`, если результатов нет:

```go
latest, err := repo.Query(ctx).
    OrderBy("created_at", repository.Desc).
    First()
```

### Count — получить количество

```go
count, err := repo.Query(ctx).
    Where(repository.Eq("status", "active")).
    Count()
```

### Exists — проверить существование

```go
exists, err := repo.Query(ctx).
    Where(repository.Eq("email", "alice@example.com")).
    Exists()
```

---

## Keyset-пагинация

Курсорная пагинация эффективнее OFFSET для больших наборов данных. Используется подход Keyset Pagination: вместо смещения передаётся курсор, указывающий на последний элемент предыдущей страницы.

### Первая страница

```go
extractor := func(u *User) map[string]any {
    s := u.Snapshot()
    return map[string]any{"id": s.ID, "created_at": s.CreatedAt}
}

page, err := repo.Query(ctx).
    Where(repository.Eq("status", "active")).
    OrderBy("created_at", repository.Desc).
    PageSize(20).
    Page(extractor)
```

### Результат

Структура `Page[T]`:

```go
type Page[T any] struct {
    Items      []T    `json:"items"`
    NextCursor string `json:"next_cursor,omitempty"`
    HasMore    bool   `json:"has_more"`
}
```

### Следующие страницы

```go
page, err := repo.Query(ctx).
    Where(repository.Eq("status", "active")).
    OrderBy("created_at", repository.Desc).
    PageSize(20).
    After(previousPage.NextCursor).   // курсор из предыдущего ответа
    Page(extractor)
```

### Обратная навигация

```go
page, err := repo.Query(ctx).
    OrderBy("created_at", repository.Desc).
    PageSize(20).
    Before(cursor).
    Page(extractor)
```

### Как работает курсор

Курсор — это Base64-кодированный JSON с значениями колонок сортировки последнего элемента. При следующем запросе эти значения используются для построения Keyset-условия (`created_at < $1 OR (created_at = $1 AND id > $2)`).

Все колонки первичного ключа автоматически добавляются в `ORDER BY`, если отсутствуют, для гарантии детерминированного порядка. Для составных ключей добавляются все компоненты.

### Использование в HTTP-API

```go
func ListUsersHandler(repo *repository.Repository[*User]) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        cursor := r.URL.Query().Get("cursor")
        extractor := func(u *User) map[string]any {
            return map[string]any{"id": u.ID()}
        }

        q := repo.Query(r.Context()).
            OrderBy("created_at", repository.Desc).
            PageSize(20)

        if cursor != "" {
            q = q.After(cursor)
        }

        page, err := q.Page(extractor)
        if err != nil {
            http.Error(w, err.Error(), 500)
            return
        }

        json.NewEncoder(w).Encode(page)
    }
}
```

---

## Soft Delete

При указании `SoftDelete` в `Table`:

```go
var userTable = repository.Table{
    Name:       "users",
    PrimaryKey: []string{"id"},
    Columns:    []string{"id", "name", "email"},
    SoftDelete: "deleted_at",
}
```

- `Delete` выполняет `UPDATE users SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
- `Find`, `FindBy`, `Query`, `ExistsBy`, `CountBy` автоматически добавляют `AND deleted_at IS NULL`
- Фильтр применяется прозрачно — вызывающий код не знает о soft delete

---

## Optimistic Locking

При указании `VersionColumn` в `Table`:

```go
var userTable = repository.Table{
    Name:          "users",
    PrimaryKey:    []string{"id"},
    Columns:       []string{"id", "name", "email", "version"},
    VersionColumn: "version",
}
```

- `Save` генерирует SQL с `version = version + 1` в секции UPDATE
- PostgreSQL/SQLite: добавляется `WHERE table.version = EXCLUDED.version`
- Если ни одна строка не обновлена — возвращается `ErrConcurrentModification`
- Значение `version` передаётся в `Values` текущим значением; инкремент происходит в SQL

```go
err := repo.Save(ctx, user)
if errors.Is(err, repository.ErrConcurrentModification) {
    // кто-то обновил запись параллельно — перечитайте и повторите
}
```

---

## Транзакции

### SaveTx / DeleteTx — работа с внешней транзакцией

```go
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()

if err := userRepo.SaveTx(ctx, tx, user); err != nil {
    return err
}
if err := profileRepo.SaveTx(ctx, tx, profile); err != nil {
    return err
}

return tx.Commit()
```

`DeleteTx` также поддерживает составной ключ:

```go
err := repo.DeleteTx(ctx, tx, accountID, roleID)
```

### Автоматические транзакции для Composite

При сохранении составных агрегатов с `Save` (не `SaveTx`) транзакция создаётся автоматически, если репозиторий создан с `*sql.DB`. Все дочерние операции выполняются в одной транзакции.

---

## Составные агрегаты (Composite)

Composite-маппинг предназначен для агрегатов, данные которых хранятся в нескольких таблицах.

### Пример: Заказ с позициями

Определение таблиц:

```go
var orderTable = repository.Table{
    Name:          "orders",
    PrimaryKey:    []string{"id"},
    Columns:       []string{"id", "customer_id", "status", "version"},
    VersionColumn: "version",
}

var orderItemsRelation = repository.Relation{
    Table:      "order_items",
    ForeignKey: "order_id",
    PrimaryKey: "item_id",
    Columns:    []string{"item_id", "order_id", "product_id", "quantity", "price"},
    OnSave:     repository.DeleteAndReinsert,
}
```

### Стратегии сохранения дочерних записей

| Стратегия | Описание |
|-----------|----------|
| `DeleteAndReinsert` | Удаляет все дочерние записи по FK, затем вставляет заново batch-операцией |
| `Upsert` | Для каждой дочерней записи выполняет отдельный Upsert |

`DeleteAndReinsert` эффективнее при полной замене набора дочерних записей.
`Upsert` подходит, когда дочерние записи изменяются по одной.

### Промежуточный снимок

```go
type OrderSnapshot struct {
    ID         string
    CustomerID string
    Status     string
    Version    int
    Items      []OrderItemSnapshot
}

type OrderItemSnapshot struct {
    ItemID    string
    OrderID   string
    ProductID string
    Quantity  int
    Price     float64
}

// Restore восстанавливает агрегат из снимка
func (s *OrderSnapshot) Restore() (*Order, error) {
    return RestoreOrder(s)
}
```

### Реализация маппинга

```go
func NewOrderRepository(db *sql.DB) *repository.Repository[*Order] {
    return repository.New(db, repository.Postgres(), repository.Composite(
        repository.CompositeConfig[*Order, *OrderSnapshot]{
            Table:     orderTable,
            Relations: []repository.Relation{orderItemsRelation},

            ScanRoot: func(sc repository.Scanner) (*OrderSnapshot, error) {
                s := &OrderSnapshot{}
                err := sc.Scan(&s.ID, &s.CustomerID, &s.Status, &s.Version)
                return s, err
            },

            ScanChild: func(table string, sc repository.Scanner, snap *OrderSnapshot) error {
                switch table {
                case "order_items":
                    var item OrderItemSnapshot
                    if err := sc.Scan(
                        &item.ItemID, &item.OrderID,
                        &item.ProductID, &item.Quantity, &item.Price,
                    ); err != nil {
                        return err
                    }
                    snap.Items = append(snap.Items, item)
                }
                return nil
            },

            Build: func(snap *OrderSnapshot) (*Order, error) {
                return snap.Restore()
            },

            Decompose: func(o *Order) repository.CompositeValues {
                s := o.Snapshot()
                children := make([][]any, len(s.Items))
                for i, item := range s.Items {
                    children[i] = []any{
                        item.ItemID, item.OrderID,
                        item.ProductID, item.Quantity, item.Price,
                    }
                }
                return repository.CompositeValues{
                    Root:     []any{s.ID, s.CustomerID, s.Status, s.Version},
                    Children: map[string][][]any{"order_items": children},
                }
            },

            ExtractPK: func(snap *OrderSnapshot) string {
                return snap.ID
            },
        },
    ))
}
```

### Как работает чтение

**`findOne`:**
1. Выполняет SELECT для корневой таблицы → `ScanRoot` → снимок
2. Для каждой Relation выполняет SELECT по FK → `ScanChild` → заполняет снимок
3. `Build` → финальный агрегат

**`findMany`:**
1. Выполняет SELECT для корневой таблицы → несколько снимков
2. Собирает все PK
3. Для каждой Relation выполняет batch SELECT с `WHERE fk IN (...)` → раздаёт по снимкам
4. `Build` для каждого → срез агрегатов

При отсутствии Relations оба метода работают без дополнительных запросов.

### Как работает запись

**`save`:**
1. Если нет Relations — простой Upsert корневой строки
2. Если есть Relations — оборачивает в транзакцию (при наличии `TxBeginner`):
   - Upsert корневой строки
   - Для каждой Relation по стратегии: `DeleteAndReinsert` или `Upsert`

**`delete`:**
1. Soft Delete или нет Relations — один запрос
2. Иначе — транзакция: дочерние таблицы в обратном порядке, затем корневая запись

---

## Ошибки

Пакет определяет три sentinel-ошибки:

```go
var (
    ErrNotFound               = errors.New("entity not found")
    ErrConcurrentModification = errors.New("concurrent modification")
    ErrInvalidCursor          = errors.New("invalid cursor")
)
```

Проверка через `errors.Is`:

```go
user, err := repo.Find(ctx, "u-1")
if errors.Is(err, repository.ErrNotFound) {
    // сущность не найдена
}

err = repo.Save(ctx, user)
if errors.Is(err, repository.ErrConcurrentModification) {
    // конфликт версий — перечитайте и повторите
}

page, err := repo.Query(ctx).After("bad-cursor").Page(extractor)
if errors.Is(err, repository.ErrInvalidCursor) {
    // невалидный курсор
}
```

---

## Полный пример

Полноценный пример с PostgreSQL, Soft Delete, версионированием и пагинацией:

```go
package main

import (
    "context"
    "database/sql"
    "encoding/json"
    "errors"
    "fmt"
    "log"
    "os"

    _ "github.com/lib/pq"

    repo "github.com/shuldan/repository"
)

// --- Домен ---

type Article struct {
    id      string
    title   string
    status  string
    version int
}

func NewArticle(id, title string) *Article {
    return &Article{id: id, title: title, status: "draft", version: 1}
}

func (a *Article) ID() string      { return a.id }
func (a *Article) Title() string   { return a.title }
func (a *Article) Publish()        { a.status = "published" }

// ArticleSnapshot — плоское представление для персистентности
type ArticleSnapshot struct {
    ID, Title, Status string
    Version           int
}

// Snapshot возвращает плоский снимок агрегата
func (a *Article) Snapshot() ArticleSnapshot {
    return ArticleSnapshot{a.id, a.title, a.status, a.version}
}

// Restore восстанавливает агрегат из снимка
func (s ArticleSnapshot) Restore() *Article {
    return &Article{id: s.ID, title: s.Title, status: s.Status, version: s.Version}
}

// --- Инфраструктура ---

var articleTable = repo.Table{
    Name:          "articles",
    PrimaryKey:    []string{"id"},
    Columns:       []string{"id", "title", "status", "version"},
    VersionColumn: "version",
    SoftDelete:    "deleted_at",
    CreatedAt:     "created_at",
    UpdatedAt:     "updated_at",
}

func NewArticleRepo(db *sql.DB) *repo.Repository[*Article] {
    return repo.New(db, repo.Postgres(), repo.Simple(repo.SimpleConfig[*Article]{
        Table: articleTable,
        Scan: func(sc repo.Scanner) (*Article, error) {
            var s ArticleSnapshot
            if err := sc.Scan(&s.ID, &s.Title, &s.Status, &s.Version); err != nil {
                return nil, err
            }
            return s.Restore(), nil
        },
        Values: func(a *Article) []any {
            s := a.Snapshot()
            return []any{s.ID, s.Title, s.Status, s.Version}
        },
    }))
}

// --- Использование ---

func main() {
    db, _ := sql.Open("postgres", os.Getenv("DATABASE_URL"))
    defer db.Close()

    articles := NewArticleRepo(db)
    ctx := context.Background()

    // Создаём статью
    a := NewArticle("a-1", "Hello World")
    a.Publish()
    if err := articles.Save(ctx, a); err != nil {
        log.Fatal(err)
    }

    // Поиск по условию
    published, err := articles.FindBy(ctx, repo.Eq("status", "published"))
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Опубликовано: %d\n", len(published))

    // Пагинация
    extractor := func(a *Article) map[string]any {
        return map[string]any{"id": a.ID()}
    }
    page, err := articles.Query(ctx).
        Where(repo.Eq("status", "published")).
        OrderBy("created_at", repo.Desc).
        PageSize(10).
        Page(extractor)
    if err != nil {
        log.Fatal(err)
    }

    out, _ := json.MarshalIndent(page, "", "  ")
    fmt.Println(string(out))

    // Soft Delete
    if err := articles.Delete(ctx, "a-1"); err != nil {
        log.Fatal(err)
    }

    // Запись уже не найдётся через Find (deleted_at IS NULL автоматически)
    _, err = articles.Find(ctx, "a-1")
    if errors.Is(err, repo.ErrNotFound) {
        fmt.Println("Статья мягко удалена")
    }
}
```

---

## API-справочник

### Repository[T]

| Метод | Описание |
|-------|----------|
| `Find(ctx, ids ...any) (T, error)` | Поиск по первичному ключу (одиночному или составному) |
| `FindBy(ctx, Spec) ([]T, error)` | Поиск по спецификации |
| `ExistsBy(ctx, Spec) (bool, error)` | Проверка существования |
| `CountBy(ctx, Spec) (int64, error)` | Подсчёт записей |
| `Save(ctx, T) error` | Upsert агрегата |
| `SaveTx(ctx, *sql.Tx, T) error` | Upsert в транзакции |
| `Delete(ctx, ids ...any) error` | Удаление по первичному ключу |
| `DeleteTx(ctx, *sql.Tx, ids ...any) error` | Удаление в транзакции |
| `Query(ctx) *Query[T]` | Fluent-построитель запросов |

### Query[T]

| Метод | Описание |
|-------|----------|
| `Where(Spec)` | Добавить условие (AND) |
| `OrderBy(column, Direction)` | Добавить сортировку |
| `Limit(n)` | Ограничить количество |
| `Offset(n)` | Смещение |
| `PageSize(n)` | Размер страницы (по умолчанию 20) |
| `After(cursor)` | Курсор для следующей страницы |
| `Before(cursor)` | Курсор для предыдущей страницы |
| `All() ([]T, error)` | Все результаты |
| `First() (T, error)` | Первый результат |
| `Count() (int64, error)` | Количество |
| `Exists() (bool, error)` | Существование |
| `Page(CursorExtractor[T]) (*Page[T], error)` | Страница с курсором |

### Table

| Поле | Тип | Описание |
|------|-----|----------|
| `Name` | `string` | Имя таблицы |
| `PrimaryKey` | `[]string` | Колонки первичного ключа |
| `Columns` | `[]string` | Все колонки |
| `VersionColumn` | `string` | Колонка версии (Optimistic Locking) |
| `SoftDelete` | `string` | Колонка мягкого удаления |
| `CreatedAt` | `string` | Колонка времени создания |
| `UpdatedAt` | `string` | Колонка времени обновления |

### Спецификации

| Функция | SQL |
|---------|-----|
| `Eq(col, val)` | `col = $N` |
| `NotEq(col, val)` | `col != $N` |
| `Gt(col, val)` | `col > $N` |
| `Gte(col, val)` | `col >= $N` |
| `Lt(col, val)` | `col < $N` |
| `Lte(col, val)` | `col <= $N` |
| `In(col, vals...)` | `col IN ($N, ...)` |
| `NotIn(col, vals...)` | `col NOT IN ($N, ...)` |
| `Like(col, pattern)` | `col LIKE $N` |
| `ILike(col, pattern)` | `col ILIKE $N` |
| `Between(col, from, to)` | `col BETWEEN $N AND $M` |
| `IsNull(col)` | `col IS NULL` |
| `IsNotNull(col)` | `col IS NOT NULL` |
| `And(specs...)` | `(...) AND (...)` |
| `Or(specs...)` | `(...) OR (...)` |
| `Not(spec)` | `NOT (...)` |
| `Raw(sql, args...)` | произвольный SQL |

---

## Разработка

### Команды

```sh
make all             # fmt + lint + security + test
make ci              # CI-проверка
make test            # Быстрые тесты
make test-coverage   # Тесты с отчётом о покрытии (> 90%)
make fmt             # Автоформатирование
make fmt-check       # Проверка форматирования
make lint            # golangci-lint
make security        # gosec
```

### Установка инструментов

```sh
make install-tools
```

Устанавливает: `golangci-lint` (v2.4.0), `goimports`, `gosec`.

### Тестовое покрытие

```sh
make test-coverage
open coverage.html
```

Текущее покрытие: **>97%**.

### Требования к PR

- Покрытие тестами новой функциональности
- Соответствие `golangci-lint`
- Без внешних зависимостей

---

## Лицензия

[MIT](LICENSE)

---

## Полезные ссылки

- [Repository Pattern](https://martinfowler.com/eaaCatalog/repository.html) — Martin Fowler
- [Keyset Pagination](https://use-the-index-luke.com/no-offset) — Markus Winand
- [Optimistic Offline Lock](https://martinfowler.com/eaaCatalog/optimisticOfflineLock.html)
- [Go Generics](https://go.dev/doc/tutorial/generics)
