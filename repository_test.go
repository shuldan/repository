package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

type testID string

func (t testID) String() string {
	return string(t)
}

type testAggregate struct {
	id testID
}

func (t *testAggregate) ID() ID {
	return t.id
}

type mockMapper struct {
	findRow                *sql.Row
	findAllRows            *sql.Rows
	findAllErr             error
	findByRows             *sql.Rows
	findByErr              error
	existsByResult         bool
	existsByErr            error
	countByResult          int64
	countByErr             error
	saveErr                error
	deleteErr              error
	fromRowAggregate       *testAggregate
	fromRowErr             error
	fromRowsResult         []*testAggregate
	fromRowsErr            error
	findCalled             int
	findAllCalled          int
	findByCalled           int
	existsByCalled         int
	countByCalled          int
	saveCalled             int
	deleteCalled           int
	fromRowCalled          int
	fromRowsCalled         int
	lastFindID             ID
	lastDeleteID           ID
	lastSaveAggregate      *testAggregate
	lastFindByConditions   string
	lastFindByArgs         []any
	lastExistsByConditions string
	lastExistsByArgs       []any
	lastCountByConditions  string
	lastCountByArgs        []any
}

func (m *mockMapper) Find(ctx context.Context, db *sql.DB, id ID) *sql.Row {
	m.findCalled++
	m.lastFindID = id
	return m.findRow
}

func (m *mockMapper) FindAll(ctx context.Context, db *sql.DB, limit, offset int) (*sql.Rows, error) {
	m.findAllCalled++
	return m.findAllRows, m.findAllErr
}

func (m *mockMapper) FindBy(ctx context.Context, db *sql.DB, conditions string, args []any) (*sql.Rows, error) {
	m.findByCalled++
	m.lastFindByConditions = conditions
	m.lastFindByArgs = args
	return m.findByRows, m.findByErr
}

func (m *mockMapper) ExistsBy(ctx context.Context, db *sql.DB, conditions string, args []any) (bool, error) {
	m.existsByCalled++
	m.lastExistsByConditions = conditions
	m.lastExistsByArgs = args
	return m.existsByResult, m.existsByErr
}

func (m *mockMapper) CountBy(ctx context.Context, db *sql.DB, conditions string, args []any) (int64, error) {
	m.countByCalled++
	m.lastCountByConditions = conditions
	m.lastCountByArgs = args
	return m.countByResult, m.countByErr
}

func (m *mockMapper) Save(ctx context.Context, db *sql.DB, aggregate *testAggregate) error {
	m.saveCalled++
	m.lastSaveAggregate = aggregate
	return m.saveErr
}

func (m *mockMapper) Delete(ctx context.Context, db *sql.DB, id ID) error {
	m.deleteCalled++
	m.lastDeleteID = id
	return m.deleteErr
}

func (m *mockMapper) FromRow(row *sql.Row) (*testAggregate, error) {
	m.fromRowCalled++
	return m.fromRowAggregate, m.fromRowErr
}

func (m *mockMapper) FromRows(rows *sql.Rows) ([]*testAggregate, error) {
	m.fromRowsCalled++
	return m.fromRowsResult, m.fromRowsErr
}

func TestNewRepository_Success(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	mapper := &mockMapper{}

	repo := NewRepository[*testAggregate, testID](db, mapper)

	if repo == nil {
		t.Fatal("NewRepository returned nil")
	}
}

func TestRepository_Find_Success(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	agg := &testAggregate{id: "test-id"}
	mapper := &mockMapper{
		findRow:          &sql.Row{},
		fromRowAggregate: agg,
	}

	repo := NewRepository[*testAggregate, testID](db, mapper)
	result, err := repo.Find(context.Background(), testID("test-id"))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID().String() != "test-id" {
		t.Errorf("expected id 'test-id', got %v", result.ID().String())
	}

	if mapper.findCalled != 1 {
		t.Errorf("expected Find called 1 time, got %d", mapper.findCalled)
	}
}

func TestRepository_Find_NotFound(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	mapper := &mockMapper{
		findRow:    &sql.Row{},
		fromRowErr: sql.ErrNoRows,
	}

	repo := NewRepository[*testAggregate, testID](db, mapper)
	_, err := repo.Find(context.Background(), testID("missing-id"))

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrEntityNotFound) {
		t.Errorf("expected ErrEntityNotFound, got %v", err)
	}

	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows in chain, got %v", err)
	}
}

func TestRepository_Find_FromRowError(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	expectedErr := errors.New("from row error")
	mapper := &mockMapper{
		findRow:    &sql.Row{},
		fromRowErr: expectedErr,
	}

	repo := NewRepository[*testAggregate, testID](db, mapper)
	_, err := repo.Find(context.Background(), testID("id"))

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
}

func TestRepository_FindAll_Success(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	agg1 := &testAggregate{id: "id1"}
	agg2 := &testAggregate{id: "id2"}

	mapper := &mockMapper{
		findAllRows:    &sql.Rows{},
		fromRowsResult: []*testAggregate{agg1, agg2},
	}

	repo := NewRepository[*testAggregate, testID](db, mapper)
	results, err := repo.FindAll(context.Background(), 10, 0)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestRepository_FindAll_FindAllError(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	expectedErr := errors.New("find all error")
	mapper := &mockMapper{
		findAllErr: expectedErr,
	}

	repo := NewRepository[*testAggregate, testID](db, mapper)
	_, err := repo.FindAll(context.Background(), 10, 0)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
}

func TestRepository_FindAll_FromRowsError(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	expectedErr := errors.New("rows error")
	mapper := &mockMapper{
		findAllRows: &sql.Rows{},
		fromRowsErr: expectedErr,
	}

	repo := NewRepository[*testAggregate, testID](db, mapper)
	_, err := repo.FindAll(context.Background(), 10, 0)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
}

func TestRepository_FindBy_Success(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	agg := &testAggregate{id: "id"}

	mapper := &mockMapper{
		findByRows:     &sql.Rows{},
		fromRowsResult: []*testAggregate{agg},
	}

	repo := NewRepository[*testAggregate, testID](db, mapper)
	results, err := repo.FindBy(context.Background(), "status = ?", []any{"active"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	if mapper.lastFindByConditions != "status = ?" {
		t.Errorf("expected 'status = ?', got %v", mapper.lastFindByConditions)
	}
}

func TestRepository_FindBy_FindByError(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	expectedErr := errors.New("find by error")
	mapper := &mockMapper{
		findByErr: expectedErr,
	}

	repo := NewRepository[*testAggregate, testID](db, mapper)
	_, err := repo.FindBy(context.Background(), "status = ?", []any{"active"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
}

func TestRepository_FindBy_FromRowsError(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	expectedErr := errors.New("rows error")
	mapper := &mockMapper{
		findByRows:  &sql.Rows{},
		fromRowsErr: expectedErr,
	}

	repo := NewRepository[*testAggregate, testID](db, mapper)
	_, err := repo.FindBy(context.Background(), "x=?", []any{1})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
}

func TestRepository_ExistsBy_True(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	mapper := &mockMapper{
		existsByResult: true,
	}

	repo := NewRepository[*testAggregate, testID](db, mapper)
	exists, err := repo.ExistsBy(context.Background(), "id = ?", []any{"123"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !exists {
		t.Error("expected true, got false")
	}

	if mapper.lastExistsByConditions != "id = ?" {
		t.Errorf("expected 'id = ?', got %v", mapper.lastExistsByConditions)
	}
}

func TestRepository_ExistsBy_False(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	mapper := &mockMapper{
		existsByResult: false,
	}

	repo := NewRepository[*testAggregate, testID](db, mapper)
	exists, err := repo.ExistsBy(context.Background(), "id = ?", []any{"999"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if exists {
		t.Error("expected false, got true")
	}
}

func TestRepository_ExistsBy_Error(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	expectedErr := errors.New("exists error")
	mapper := &mockMapper{
		existsByErr: expectedErr,
	}

	repo := NewRepository[*testAggregate, testID](db, mapper)
	_, err := repo.ExistsBy(context.Background(), "x=?", []any{1})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
}

func TestRepository_CountBy_Success(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	mapper := &mockMapper{
		countByResult: 42,
	}

	repo := NewRepository[*testAggregate, testID](db, mapper)
	count, err := repo.CountBy(context.Background(), "active = ?", []any{true})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count != 42 {
		t.Errorf("expected 42, got %d", count)
	}

	if mapper.lastCountByConditions != "active = ?" {
		t.Errorf("expected 'active = ?', got %v", mapper.lastCountByConditions)
	}
}

func TestRepository_CountBy_Zero(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	mapper := &mockMapper{
		countByResult: 0,
	}

	repo := NewRepository[*testAggregate, testID](db, mapper)
	count, err := repo.CountBy(context.Background(), "x=?", []any{1})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

func TestRepository_CountBy_Error(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	expectedErr := errors.New("count error")
	mapper := &mockMapper{
		countByErr: expectedErr,
	}

	repo := NewRepository[*testAggregate, testID](db, mapper)
	_, err := repo.CountBy(context.Background(), "x=?", []any{1})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
}

func TestRepository_Save_Success(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	agg := &testAggregate{id: "save-id"}
	mapper := &mockMapper{}

	repo := NewRepository[*testAggregate, testID](db, mapper)
	err := repo.Save(context.Background(), agg)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mapper.saveCalled != 1 {
		t.Errorf("expected Save called 1 time, got %d", mapper.saveCalled)
	}

	if mapper.lastSaveAggregate == nil {
		t.Fatal("expected aggregate to be saved")
	}

	if mapper.lastSaveAggregate.ID().String() != "save-id" {
		t.Errorf("expected 'save-id', got %v", mapper.lastSaveAggregate.ID().String())
	}
}

func TestRepository_Save_Error(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	expectedErr := errors.New("save error")
	agg := &testAggregate{id: "id"}
	mapper := &mockMapper{
		saveErr: expectedErr,
	}

	repo := NewRepository[*testAggregate, testID](db, mapper)
	err := repo.Save(context.Background(), agg)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
}

func TestRepository_Delete_Success(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	mapper := &mockMapper{}

	repo := NewRepository[*testAggregate, testID](db, mapper)
	err := repo.Delete(context.Background(), testID("delete-id"))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mapper.deleteCalled != 1 {
		t.Errorf("expected Delete called 1 time, got %d", mapper.deleteCalled)
	}

	if mapper.lastDeleteID.String() != "delete-id" {
		t.Errorf("expected 'delete-id', got %v", mapper.lastDeleteID.String())
	}
}

func TestRepository_Delete_Error(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	expectedErr := errors.New("delete error")
	mapper := &mockMapper{
		deleteErr: expectedErr,
	}

	repo := NewRepository[*testAggregate, testID](db, mapper)
	err := repo.Delete(context.Background(), testID("id"))

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
}

func TestRepository_FindAll_EmptyResult(t *testing.T) {
	t.Parallel()

	db := &sql.DB{}
	mapper := &mockMapper{
		findAllRows:    &sql.Rows{},
		fromRowsResult: []*testAggregate{},
	}

	repo := NewRepository[*testAggregate, testID](db, mapper)
	results, err := repo.FindAll(context.Background(), 10, 0)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected empty result, got %d items", len(results))
	}
}

func TestRepository_IntegrationScenarios(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupRepo func() (Repository[*testAggregate, testID], *mockMapper)
		operation func(Repository[*testAggregate, testID]) error
		wantErr   bool
	}{
		{
			name: "find_all_with_limit_offset",
			setupRepo: func() (Repository[*testAggregate, testID], *mockMapper) {
				db := &sql.DB{}
				agg := &testAggregate{id: "1"}
				mapper := &mockMapper{findAllRows: &sql.Rows{}, fromRowsResult: []*testAggregate{agg}}
				return NewRepository[*testAggregate, testID](db, mapper), mapper
			},
			operation: func(r Repository[*testAggregate, testID]) error {
				_, err := r.FindAll(context.Background(), 100, 50)
				return err
			},
			wantErr: false,
		},
		{
			name: "find_by_with_multiple_args",
			setupRepo: func() (Repository[*testAggregate, testID], *mockMapper) {
				db := &sql.DB{}
				agg := &testAggregate{id: "x"}
				mapper := &mockMapper{findByRows: &sql.Rows{}, fromRowsResult: []*testAggregate{agg}}
				return NewRepository[*testAggregate, testID](db, mapper), mapper
			},
			operation: func(r Repository[*testAggregate, testID]) error {
				_, err := r.FindBy(context.Background(), "a=? AND b=?", []any{1, 2})
				return err
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, _ := tt.setupRepo()
			err := tt.operation(repo)
			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got err=%v", tt.wantErr, err)
			}
		})
	}
}
