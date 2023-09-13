package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {

	g := gin.Default()
	g.GET("/", func(ctx *gin.Context) {

		ctx.JSON(http.StatusOK, map[string]any{"data": "hello world!"})
	})

	if err := g.Run(); err != nil {
		log.Fatal("run server failed: ", err)
	}
}
