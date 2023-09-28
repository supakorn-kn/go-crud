package books

import (
	"context"
	"fmt"
	"slices"

	"github.com/supakorn-kn/go-crud/errors"
	"github.com/supakorn-kn/go-crud/models"
	"github.com/supakorn-kn/go-crud/mongodb"
	"github.com/supakorn-kn/go-crud/objects"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SearchOption struct {
	CurrentPage int                `json:"current_page"`
	Title       models.MatchOption `json:"title,omitempty"`
	Author      models.MatchOption `json:"author,omitempty"`
	Categories  []string           `json:"categories,omitempty"`
}

type BooksModel struct {
	models.BaseModel[objects.Book]
}

func NewBooksModel(conn *mongodb.MongoDBConn, paginateSize ...int) (*BooksModel, error) {

	var searchSize int = 10

	paginateSizeLen := len(paginateSize)
	if paginateSizeLen > 1 {
		return nil, fmt.Errorf("PaginateSize can have only one elements")
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

	err := m.initCollection(conn)
	if err != nil {
		return err
	}

	err = m.initIndexes(conn)
	if err != nil {
		return err
	}

	coll := conn.GetDatabase().Collection(m.GetCollectionName())

	baseModel, err := models.NewBaseModel[objects.Book](coll, paginateSize, "book_id")
	if err != nil {
		return err
	}

	m.BaseModel = *baseModel

	return nil
}

func (m BooksModel) initCollection(conn *mongodb.MongoDBConn) error {

	bookDB := conn.GetDatabase()
	collectionName := m.GetCollectionName()

	filter := bson.D{}
	option := options.ListCollections()
	collectionNameList, err := bookDB.ListCollectionNames(context.Background(), filter, option)
	if err != nil {
		return err
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
			{Key: "collMod", Value: "books_info"},
			{Key: "validator", Value: validator},
			{Key: "validationLevel", Value: "strict"},
		}

		option := options.RunCmd()
		result := bookDB.RunCommand(context.Background(), cmd, option)
		if err := result.Err(); err != nil {
			return err
		}

		return nil
	}

	collectionOption := options.CreateCollection()
	collectionOption.SetValidator(validator)
	collectionOption.SetValidationLevel("strict")

	err = bookDB.CreateCollection(context.Background(), collectionName, collectionOption)
	if err != nil {
		return err
	}

	return nil
}

func (m BooksModel) initIndexes(conn *mongodb.MongoDBConn) error {

	collectionName := m.GetCollectionName()
	coll := conn.GetDatabase().Collection(collectionName)
	cur, err := coll.Indexes().List(context.Background())
	if err != nil {
		return nil
	}

	var titleAndAuthorIndexName = "title_1_author_1"
	var bookIDIndex = "book_id_1"

	var indexes []bson.M
	err = cur.All(context.Background(), &indexes)
	if err != nil {
		return err
	}

	contains := slices.ContainsFunc(indexes, func(m primitive.M) bool {
		return m["name"] == titleAndAuthorIndexName
	})

	if !contains {

		indexModelOption := options.Index()
		indexModelOption.SetName(titleAndAuthorIndexName)

		indexModel := mongo.IndexModel{
			Keys: bson.D{
				{Key: "title", Value: 1},
				{Key: "author", Value: 1},
			},
			Options: indexModelOption,
		}

		option := options.CreateIndexes()
		_, err = coll.Indexes().CreateOne(context.Background(), indexModel, option)
		if err != nil {
			return err
		}
	}

	contains = slices.ContainsFunc(indexes, func(m primitive.M) bool {
		return m["name"] == bookIDIndex
	})

	if !contains {

		indexModelOption := options.Index()
		indexModelOption.SetName(bookIDIndex)
		indexModelOption.SetUnique(true)

		indexModel := mongo.IndexModel{
			Keys: bson.D{
				{Key: "book_id", Value: 1},
			},
			Options: indexModelOption,
		}

		option := options.CreateIndexes()
		_, err = coll.Indexes().CreateOne(context.Background(), indexModel, option)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m BooksModel) Search(opt SearchOption) (paginationData models.PaginationData[objects.Book], paginationErr error) {

	if opt.CurrentPage < 1 {
		paginationErr = errors.CurrentPageInvalidError.New()
		return
	}

	matchConditions := bson.A{}
	if !opt.Title.IsNil() {

		matchBson, err := models.CreateMatchBson("title", opt.Title.Value, opt.Title.MatchType)
		if err != nil {
			paginationErr = err
			return
		}

		matchConditions = append(matchConditions, matchBson)
	}

	if !opt.Author.IsNil() {

		matchBson, err := models.CreateMatchBson("author", opt.Author.Value, opt.Author.MatchType)
		if err != nil {
			paginationErr = err
			return
		}

		matchConditions = append(matchConditions, matchBson)
	}

	if opt.Categories != nil {
		cond := bson.D{{Key: "categories", Value: bson.D{{Key: "$in", Value: opt.Categories}}}}
		matchConditions = append(matchConditions, cond)
	}

	if len(matchConditions) == 0 {
		matchConditions = append(matchConditions, bson.D{})
	}

	matchStage := bson.D{
		{
			Key: "$match", Value: bson.D{
				{Key: "$and", Value: matchConditions},
			},
		},
	}

	paginateResultQuery := bson.A{
		bson.D{{Key: "$sort", Value: bson.D{{Key: "title", Value: 1}, {Key: "author", Value: 1}}}},
		bson.D{{Key: "$skip", Value: (opt.CurrentPage - 1) * m.BaseModel.SearchLenLimit}},
		bson.D{{Key: "$limit", Value: m.BaseModel.SearchLenLimit}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "data", Value: bson.D{{Key: "$push", Value: "$$ROOT"}}},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}

	matchResultQuery := bson.A{
		bson.D{{Key: "$count", Value: "total"}},
	}

	facetStage := bson.D{
		{
			Key: "$facet", Value: bson.D{
				{Key: "paginate_result", Value: paginateResultQuery},
				{Key: "match_result", Value: matchResultQuery},
			},
		},
	}

	projectStage := bson.D{
		{
			Key: "$project", Value: bson.D{
				{Key: "count", Value: bson.D{{Key: "$first", Value: "$paginate_result.count"}}},
				{Key: "total", Value: bson.D{{Key: "$first", Value: "$match_result.total"}}},
				{Key: "data", Value: bson.D{{Key: "$first", Value: "$paginate_result.data"}}},
			},
		},
	}

	paginationData, paginationErr = m.BaseModel.Search(models.SearchOption{
		CurrentPage: opt.CurrentPage,
		Pipeline:    mongo.Pipeline{matchStage, facetStage, projectStage},
	})

	return
}
