package postgres

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	gstore "github.com/fharding1/todo/store"
	"github.com/stretchr/testify/assert"
)

var globalStore gstore.Service
var rawDB *sql.DB

var options = Options{
	User: os.Getenv("POSTGRES_1_ENV_POSTGRES_USER"), Pass: os.Getenv("POSTGRES_1_ENV_POSTGRES_PASSWORD"),
	Host: os.Getenv("POSTGRES_1_PORT_5432_TCP_ADDR"), Port: 5432,
	DBName: os.Getenv("POSTGRES_1_ENV_POSTGRES_DB"), SSLMode: "disable",
}

func newInt64(x int64) *int64 {
	return &x
}

var todoCases = []struct {
	initialTodo gstore.Todo
	updatedTodo gstore.Todo
}{
	{
		gstore.Todo{Description: "build this app", CreatedAt: time.Unix(1000, 0).Unix()},
		gstore.Todo{Description: "finish building this app", CreatedAt: time.Unix(2000, 0).Unix()},
	},
	{
		gstore.Todo{Description: "do homework", CreatedAt: time.Unix(10000, 0).Unix()},
		gstore.Todo{Description: "do math homework", CreatedAt: time.Unix(10000, 0).Unix(), CompletedAt: newInt64(time.Unix(20000, 0).Unix())},
	},
}

func TestMain(m *testing.M) {
	var err error
	globalStore, err = New(options)
	if err != nil {
		fmt.Printf("unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	if s, ok := globalStore.(*store); ok {
		rawDB = s.db
	} else {
		fmt.Printf("unable to cast store service to postgres service: %v\n", err)
		os.Exit(2)
	}

	os.Exit(m.Run())
}

func TestCreateTodo(t *testing.T) {
	_, err := rawDB.Exec("DELETE FROM todos")
	if err != nil {
		t.Errorf("clearing todos table: %v\n", err)
		t.FailNow()
	}

	for _, tt := range todoCases {
		id, err := globalStore.CreateTodo(tt.initialTodo)
		assert.Nil(t, err)

		var todo gstore.Todo
		err = rawDB.QueryRow(
			"SELECT description, createdAt, completedAt FROM todos WHERE id = $1", id).Scan(
			&todo.Description, &todo.CreatedAt, &todo.CompletedAt)
		assert.Nil(t, err)

		assert.Equal(t, tt.initialTodo, todo)
	}
}

func TestGetTodo(t *testing.T) {
	_, err := rawDB.Exec("DELETE FROM todos")
	if err != nil {
		t.Errorf("clearing todos table: %v\n", err)
		t.FailNow()
	}

	for _, tt := range todoCases {
		todo := tt.initialTodo

		var id int64
		err := rawDB.QueryRow(
			"INSERT INTO todos (description, createdAt, completedAt) VALUES ($1, $2, $3) RETURNING id",
			todo.Description, todo.CreatedAt, todo.CompletedAt).Scan(&id)
		assert.Nil(t, err)

		todo.ID = id

		gotTodo, err := globalStore.GetTodo(id)
		assert.Nil(t, err)
		assert.Equal(t, todo, gotTodo)
	}
}

func roughTodoEquality(t1 gstore.Todo, t2 gstore.Todo) bool {
	return t1.Description == t2.Description && t1.CreatedAt == t2.CreatedAt && t1.CompletedAt == t2.CompletedAt
}

func TestGetTodos(t *testing.T) {
	_, err := rawDB.Exec("DELETE FROM todos")
	if err != nil {
		t.Errorf("clearing todos table: %v\n", err)
		t.FailNow()
	}

	for _, tt := range todoCases {
		todo := tt.initialTodo

		_, err := rawDB.Exec(
			"INSERT INTO todos (description, createdAt, completedAt) VALUES ($1, $2, $3)",
			todo.Description, todo.CreatedAt, todo.CompletedAt)
		assert.Nil(t, err)
	}

	todos, err := globalStore.GetTodos()
	assert.Nil(t, err)

	for _, tt := range todoCases {
		expectedTodo := tt.initialTodo
		found := false

		for _, todo := range todos {
			if roughTodoEquality(expectedTodo, todo) {
				found = true
			}
		}

		assert.True(t, found)
	}
}

func TestUpdateTodo(t *testing.T) {
	_, err := rawDB.Exec("DELETE FROM todos")
	if err != nil {
		t.Errorf("clearing todos table: %v\n", err)
		t.FailNow()
	}

	for _, tt := range todoCases {
		initialTodo, updatedTodo := tt.initialTodo, tt.updatedTodo

		var id int64
		err := rawDB.QueryRow(
			"INSERT INTO todos (description, createdAt, completedAt) VALUES ($1, $2, $3) RETURNING id",
			initialTodo.Description, initialTodo.CreatedAt, initialTodo.CompletedAt).Scan(&id)
		assert.Nil(t, err)

		updatedTodo.ID = id

		err = globalStore.UpdateTodo(updatedTodo)
		assert.Nil(t, err)

		var gotTodo gstore.Todo
		err = rawDB.QueryRow(
			"SELECT description, createdAt, completedAt FROM todos WHERE id = $1", id).Scan(
			&gotTodo.Description, &gotTodo.CreatedAt, &gotTodo.CompletedAt)
		assert.Nil(t, err)

		gotTodo.ID = id

		assert.Equal(t, updatedTodo, gotTodo)
	}
}

func TestDeleteTodo(t *testing.T) {
	_, err := rawDB.Exec("DELETE FROM todos")
	if err != nil {
		t.Errorf("clearing todos table: %v\n", err)
		t.FailNow()
	}

	for _, tt := range todoCases {
		todo := tt.initialTodo

		var id int64
		err := rawDB.QueryRow(
			"INSERT INTO todos (description, createdAt, completedAt) VALUES ($1, $2, $3) RETURNING id",
			todo.Description, todo.CreatedAt, todo.CompletedAt).Scan(&id)
		assert.Nil(t, err)

		todo.ID = id

		err = globalStore.DeleteTodo(id)
		assert.Nil(t, err)

		var desc string
		err = rawDB.QueryRow("SELECT FROM todos WHERE id = $1", id).Scan(&desc)
		assert.Equal(t, sql.ErrNoRows, err)
	}
}