package main

import (
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/supakorn-kn/go-crud/apis"
	booksAPI "github.com/supakorn-kn/go-crud/apis/books"
	usersAPI "github.com/supakorn-kn/go-crud/apis/users"
	"github.com/supakorn-kn/go-crud/mongodb"
	"github.com/supakorn-kn/go-crud/objects"
)

func main() {

	//slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	//TODO: Will set parameter from env
	conn := mongodb.New("mongodb://localhost:27017", "go-crud_data")
	if err := conn.Connect(); err != nil {
		slog.Error("Create MongoDB connection failed", err)
		return
	}

	defer conn.Disconnect()

	newbooksAPI, err := booksAPI.NewBooksAPI(&conn)
	if err != nil {
		slog.Error("Create books model failed", err)
		return
	}

	newUsersAPI, err := usersAPI.NewUsersAPI(&conn)
	if err != nil {
		slog.Error("Create books model failed", err)
		return
	}

	g := gin.Default()
	apis.RegisterCrudAPI[objects.Book](newbooksAPI, g.Group("api/books"))
	apis.RegisterCrudAPI[objects.User](newUsersAPI, g.Group("api/users"))

	//TODO: Will set addr parameter from env
	if err := g.Run(); err != nil {
		slog.Error("run server failed", err)
		return
	}
}
