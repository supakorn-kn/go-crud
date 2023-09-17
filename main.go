package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/supakorn-kn/go-crud/mongodb"
)

func main() {

	//TODO: Will set parameter from env
	conn := mongodb.New("mongodb://localhost:27017", "go-crud_data")
	err := conn.Connect()
	if err != nil {
		log.Fatal(err)
	}

	defer conn.Disconnect()

	g := gin.Default()
	g.GET("/", func(ctx *gin.Context) {

		ctx.JSON(http.StatusOK, map[string]any{"data": "hello world!"})
	})

	//TODO: Will set addr parameter from env
	if err := g.Run(); err != nil {
		log.Fatal("run server failed: ", err)
	}
}
