package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/supakorn-kn/go-crud/models"
	"github.com/supakorn-kn/go-crud/mongodb"
)

func main() {

	conn, err := mongodb.InitConnection("mongodb://localhost:27017", "go-crud_data")
	if err != nil {
		log.Fatal(err)
	}

	bookModel := models.BooksModel{}
	err = bookModel.Init(conn)
	if err != nil {
		log.Fatal(err)
	}

	// for count := 0; count < 25; count++ {

	// 	v := strconv.Itoa(count)
	// 	v2 := strconv.Itoa(count + 25)

	// 	err := bookModel.Create(models.Book{
	// 		BookID:      "book_" + v2,
	// 		Title:       v2,
	// 		Author:      v,
	// 		Description: v,
	// 		Categories:  []string{v, v + "1"},
	// 	})
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// }

	result, err := bookModel.Paginate(
		models.PaginateOption{
			CurrentPage: 1,
			Title:       "0",
			Author:      "0",
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	b, _ := json.Marshal(result)
	fmt.Printf("%s\n", string(b))

	g := gin.Default()
	g.GET("/", func(ctx *gin.Context) {

		ctx.JSON(http.StatusOK, map[string]any{"data": "hello world!"})
	})

	if err := g.Run(); err != nil {
		log.Fatal("run server failed: ", err)
	}
}
