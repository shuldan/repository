# `repository` — Реализация паттерна Repository для Go-приложений

[![Go CI](https://github.com/shuldan/repository/workflows/Go%20CI/badge.svg)](https://github.com/shuldan/repository/actions)
[![codecov](https://codecov.io/gh/shuldan/repository/branch/main/graph/badge.svg)](https://codecov.io/gh/shuldan/repository)
[![Go Report Card](https://goreportcard.com/badge/github.com/shuldan/repository)](https://goreportcard.com/report/github.com/shuldan/repository)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

Этот пакет предоставляет реализацию паттерна Repository с поддержкой Memento для Go-приложений. Он обеспечивает абстракцию доступа к данным, позволяя легко интегрировать доменные объекты с реляционной базой данных.

---

## 🚀 Основные возможности

- **Паттерн Repository**: унифицированный интерфейс для доступа к данным.
- **Memento Pattern**: безопасное преобразование между доменными агрегатами и DTO для хранения.
- **Обобщения (Generics)**: типобезопасная реализация для любых агрегатов.
- **Контракты на уровне домена**: чёткие интерфейсы `Aggregate`, `ID`, `Memento`.
- **Поддержка CRUD операций**: `Find`, `FindAll`, `FindBy`, `ExistsBy`, `CountBy`, `Save`, `Delete`.

---

## 📦 Установка зависимостей и инструментов

Для работы с проектом убедитесь, что у вас установлен Go 1.24+.

Установите необходимые инструменты:

```sh
make install-tools
```

Это установит:
- `golangci-lint` (v2.4.0)
- `goimports`
- `gosec`

---

## 🛠️ Работа с проектом

### Запуск локальной проверки

```sh
make all
```

Выполняет:
- проверку форматирования кода,
- статический анализ (`golangci-lint`),
- security-сканирование (`gosec`),
- запуск тестов.

### Проверка в CI

```sh
make ci
```

Запускает:
- форматирование,
- `go vet`,
- линтер,
- тесты с отчётом о покрытии.

### Форматирование кода

```sh
make fmt
```

Автоматически форматирует `.go` файлы и сортирует импорты.

### Запуск тестов

```sh
make test
make test-coverage
```

---

## 🧱 Архитектура

### `ID`, `Aggregate`, `Memento`

Базовые контракты для доменных объектов:

- `ID`: интерфейс уникального идентификатора.
- `Aggregate`: интерфейс корня агрегата, включает `ID()` и `CreateMemento()`.
- `Memento`: интерфейс снимка состояния агрегата, включает `ID()` и `RestoreAggregate()`.

### `Mapper[T, M]`

Интерфейс, отвечающий за преобразование между SQL-запросами/результатами и Memento. Реализация маппера лежит на разработчике и зависит от конкретной схемы БД.

### `Repository[T, I]`

Основной интерфейс репозитория, объединяющий `Finder` и `Writer`:

```go
type Repository[T Aggregate, I ID] interface {
    Finder[T, I]
    Writer[T, I]
}
```

- `Finder`: методы для поиска и подсчёта.
- `Writer`: методы для сохранения и удаления.

---

## 🧪 Пример использования

```go
package main

import (
	"context"
	"database/sql"

	"github.com/shuldan/repository"
)

// Пример доменного агрегата
type User struct {
	id   userID
	name string
}

func (u *User) ID() repository.ID {
	return u.id
}

func (u *User) CreateMemento() (repository.Memento, error) {
	// Создание снимка состояния агрегата
	return &userMemento{ID: u.id, Name: u.name}, nil
}

// Пример Memento
type userMemento struct {
	ID   userID
	Name string
}

func (m *userMemento) ID() repository.ID {
	return m.ID
}

func (m *userMemento) RestoreAggregate() (repository.Aggregate, error) {
	// Восстановление агрегата из снимка
	return &User{id: m.ID, name: m.Name}, nil
}

// Пример ID
type userID string

func (u userID) String() string {
	return string(u)
}

// Пример маппера
type userMapper struct{}

func (m *userMapper) Find(ctx context.Context, db *sql.DB, id repository.ID) *sql.Row {
	// Реализация запроса
	return nil
}

// ... остальные методы Mapper[T, M] ...

func main() {
	db, _ := sql.Open("sqlite3", ":memory:")
	mapper := &userMapper{}
	repo := repository.NewRepository[*User, userID, *userMemento](db, mapper)

	ctx := context.Background()
	user, err := repo.Find(ctx, userID("123"))
	if err != nil {
		// Обработка ошибки
	}
	_ = user
}
```

---

## 📄 Лицензия

Проект распространяется под лицензией [MIT](LICENSE).

---

## 🤝 Вклад в проект

PR и issue приветствуются! Следуйте стандартам форматирования и покрывайте новый код тестами.

---

> **Автор**: MSeytumerov  
> **Репозиторий**: `github.com/shuldan/repository`  
> **Go**: `1.24.2`