package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/supakorn-kn/go-crud/apis"
	booksAPI "github.com/supakorn-kn/go-crud/apis/books"
	usersAPI "github.com/supakorn-kn/go-crud/apis/users"
	"github.com/supakorn-kn/go-crud/env"
	"github.com/supakorn-kn/go-crud/mongodb"
	"github.com/supakorn-kn/go-crud/objects"
)

func main() {

	//slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	envConfig, err := env.GetEnv()
	if err != nil {
		slog.Error("getting env failed", err)
		return
	}

	conn, err := mongodb.New(envConfig.MongoDB)
	if err != nil {
		slog.Error("create MongoDB connection failed", err)
		return
	}
	if err := conn.Connect(); err != nil {
		slog.Error("connecting to MongoDB failed", err)
		return
	}

	defer conn.Disconnect()

	newbooksAPI, err := booksAPI.NewBooksAPI(conn)
	if err != nil {
		slog.Error("create books model failed", err)
		return
	}

	newUsersAPI, err := usersAPI.NewUsersAPI(conn)
	if err != nil {
		slog.Error("create books model failed", err)
		return
	}

	g := gin.Default()
	apis.RegisterCrudAPI[objects.Book](newbooksAPI, g.Group("api/books"))
	apis.RegisterCrudAPI[objects.User](newUsersAPI, g.Group("api/users"))

	if err := g.Run(fmt.Sprintf(":%d", envConfig.Server.Port)); err != nil {
		slog.Error("run server failed", err)
		return
	}
}
