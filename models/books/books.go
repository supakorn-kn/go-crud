package books

import (
	"context"
	"errors"
	"slices"
	"strings"

	serverError "github.com/supakorn-kn/go-crud/errors"
	"github.com/supakorn-kn/go-crud/models"
	"github.com/supakorn-kn/go-crud/mongodb"
	"github.com/supakorn-kn/go-crud/objects"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SearchOptions struct {
	CurrentPage int                 `json:"current_page"`
	Title       models.MatchOptions `json:"title,omitempty"`
	Author      models.MatchOptions `json:"author,omitempty"`
	Categories  []string            `json:"categories,omitempty"`
}

type BooksModel struct {
	models.BaseModel[objects.Book]
}

func NewBooksModel(conn *mongodb.MongoDBConn, paginateSize ...int) (*BooksModel, error) {

	var searchSize int = 10

	paginateSizeLen := len(paginateSize)
	if paginateSizeLen > 1 {
		return nil, errors.New("PaginateSize can have only one elements")
	} else if paginateSizeLen == 1 {
		searchSize = paginateSize[0]
	}

	booksModel := BooksModel{}
	err := booksModel.init(conn, searchSize)
	if err != nil {
		return nil, err
	}

	return &booksModel, nil
}

func (m BooksModel) GetCollectionName() string {
	return "books_info"
}

func (m *BooksModel) init(conn *mongodb.MongoDBConn, paginateSize int) error {

	coll, err := m.createCollection(conn)
	if err != nil {
		return err
	}

	err = m.initIndexes(conn)
	if err != nil {
		return err
	}

	return m.BaseModel.Inject(coll, paginateSize, "book_id")
}

func (m BooksModel) createCollection(conn *mongodb.MongoDBConn) (*mongo.Collection, error) {

	crudDB := conn.GetDatabase()
	collectionName := m.GetCollectionName()

	collectionNameList, err := crudDB.ListCollectionNames(context.Background(), bson.D{})
	if err != nil {
		return nil, err
	}

	validator := bson.D{
		{
			Key: "$jsonSchema", Value: bson.M{
				"bsonType": "object",
				"required": []string{"book_id", "title", "author", "description", "categories"},
				"properties": bson.M{
					"book_id": bson.M{
						"bsonType":    "string",
						"description": "Book ID must not be empty",
					},
					"title": bson.M{
						"bsonType":    "string",
						"description": "Title must not be empty",
					},
					"author": bson.M{
						"bsonType":    "string",
						"description": "Author must not be empty",
					},
					"description": bson.M{
						"bsonType":    "string",
						"description": "Description must not be empty",
					},
					"categories": bson.M{
						"bsonType":    "array",
						"uniqueItems": true,
						"items": bson.M{
							"bsonType": "string",
						},
						"description": "Categories must contains unique string elements and not be empty",
					},
				},
			},
		},
	}

	if slices.Contains(collectionNameList, collectionName) {

		cmd := bson.D{
			{Key: "collMod", Value: collectionName},
			{Key: "validator", Value: validator},
			{Key: "validationLevel", Value: "strict"},
		}

		result := crudDB.RunCommand(context.Background(), cmd)
		if err := result.Err(); err != nil {
			return nil, err
		}
	} else {

		collectionOptions := options.CreateCollection()
		collectionOptions.SetValidator(validator)
		collectionOptions.SetValidationLevel("strict")

		err = crudDB.CreateCollection(context.Background(), collectionName, collectionOptions)
		if err != nil {
			return nil, err
		}
	}

	return conn.GetCollection(m.GetCollectionName()), nil
}

func (m BooksModel) initIndexes(conn *mongodb.MongoDBConn) error {

	collectionName := m.GetCollectionName()
	coll := conn.GetDatabase().Collection(collectionName)
	cur, err := coll.Indexes().List(context.Background())
	if err != nil {
		return err
	}

	var bookIDIndex = "book_id_1"

	var indexes []bson.M
	err = cur.All(context.Background(), &indexes)
	if err != nil {
		return err
	}

	contains := slices.ContainsFunc(indexes, func(m primitive.M) bool {
		return m["name"] == bookIDIndex
	})

	if !contains {

		indexModelOptions := options.Index()
		indexModelOptions.SetName(bookIDIndex)
		indexModelOptions.SetUnique(true)

		indexModel := mongo.IndexModel{
			Keys: bson.D{
				{Key: "book_id", Value: 1},
			},
			Options: indexModelOptions,
		}

		options := options.CreateIndexes()
		_, err = coll.Indexes().CreateOne(context.Background(), indexModel, options)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m BooksModel) Insert(book objects.Book) error {

	if strings.EqualFold(book.BookID, "") ||
		strings.EqualFold(book.Title, "") ||
		strings.EqualFold(book.Author, "") ||
		strings.EqualFold(book.Description, "") ||
		book.Categories == nil {

		return serverError.DataValidationFailedError.New()
	}

	filter := bson.D{
		{
			Key: "$or", Value: bson.A{
				bson.D{{Key: m.ItemIDKey, Value: book.BookID}},
			},
		},
	}

	err := m.Coll.FindOne(context.Background(), filter).Err()
	if err == nil {
		return serverError.DataAlreadyInUsedError.New()
	}

	return m.BaseModel.Insert(book)
}

func (m BooksModel) Search(opt SearchOptions) (paginationData models.PaginationData[objects.Book], paginationErr error) {

	var builder = models.NewSearchPipelineBuilder()
	paginationErr = builder.SortedBy([]models.SortData{
		{
			Key:    "title",
			SortBy: models.SortASC,
		},
		{
			Key:    "author",
			SortBy: models.SortASC,
		},
	})
	if paginationErr != nil {
		return
	}

	builder.Skip((opt.CurrentPage - 1) * m.BaseModel.SearchLenLimit)
	builder.Limit(m.BaseModel.SearchLenLimit)

	if !opt.Title.IsNil() {

		paginationErr = builder.Match("title", opt.Title.Value, opt.Title.MatchType)
		if paginationErr != nil {
			return
		}
	}

	if !opt.Author.IsNil() {

		paginationErr = builder.Match("author", opt.Author.Value, opt.Author.MatchType)
		if paginationErr != nil {
			return
		}
	}

	if opt.Categories != nil || len(opt.Categories) != 0 {

		paginationErr = builder.Match("categories", opt.Categories, models.ContainsInMatchType)
		if paginationErr != nil {
			return
		}
	}

	paginationData, paginationErr = m.BaseModel.Search(models.BaseSearchOptions{
		CurrentPage: opt.CurrentPage,
		Pipeline:    builder.BuildPipeline(),
	})

	return
}
