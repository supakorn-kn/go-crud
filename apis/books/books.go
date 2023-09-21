package books

import (
	"github.com/gin-gonic/gin"
	"github.com/supakorn-kn/go-crud/models"
	"github.com/supakorn-kn/go-crud/models/books"
	"github.com/supakorn-kn/go-crud/mongodb"
	"github.com/supakorn-kn/go-crud/objects"
)

type BooksCrudAPI struct {
	model books.BooksModel
}

func NewBooksAPI(conn *mongodb.MongoDBConn) (*BooksCrudAPI, error) {

	model, err := books.NewBooksModel(conn)
	if err != nil {
		return nil, err
	}

	api := new(BooksCrudAPI)
	api.model = *model

	return api, nil
}

func (api BooksCrudAPI) Insert(ctx *gin.Context) error {

	var book objects.Book
	err := ctx.BindJSON(&book)
	if err != nil {
		return err
	}

	err = api.model.Insert(book)
	if err != nil {
		return err
	}

	return nil
}

func (api BooksCrudAPI) ReadOne(itemID string, ctx *gin.Context) (*objects.Book, error) {

	book, err := api.model.GetByID(itemID)
	if err != nil {
		return nil, err
	}

	return &book, nil
}

func (api BooksCrudAPI) Read(ctx *gin.Context) (*models.PaginationData[objects.Book], error) {

	var opt books.SearchOption
	err := ctx.BindJSON(&opt)
	if err != nil {
		return nil, err
	}

	paginationData, err := api.model.Search(opt)
	if err != nil {
		return nil, err
	}

	return &paginationData, nil
}

func (api BooksCrudAPI) Update(ctx *gin.Context) error {

	var book objects.Book
	err := ctx.BindJSON(&book)
	if err != nil {
		return err
	}

	err = api.model.Update(book)
	if err != nil {
		return err
	}

	return nil
}

func (api BooksCrudAPI) Delete(ctx *gin.Context) error {

	var book objects.Book
	err := ctx.BindJSON(&book)
	if err != nil {
		return err
	}

	err = api.model.Delete(book.BookID)
	if err != nil {
		return err
	}

	return nil
}
