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
	coll  *mongo.Collection
	limit int
}

func NewBooksModel(conn *mongodb.MongoDBConn, paginateSize ...int) (*BooksModel, error) {

	paginateSizeLen := len(paginateSize)

	if paginateSizeLen > 1 {
		return nil, fmt.Errorf("PaginateSize can have only one elements")
	}

	booksModel := BooksModel{}

	if paginateSizeLen == 0 {
		booksModel.limit = 10
	} else if paginateSize[0] < 1 {
		return nil, fmt.Errorf("PaginateSize value can be only positive integer")
	} else {
		booksModel.limit = paginateSize[0]
	}

	err := booksModel.init(conn)
	if err != nil {
		return nil, err
	}

	return &booksModel, nil
}

func (m BooksModel) GetCollectionName() string {
	return "books_info"
}

func (m *BooksModel) init(conn *mongodb.MongoDBConn) error {

	err := m.initCollection(conn)
	if err != nil {
		return err
	}

	err = m.initIndexes(conn)
	if err != nil {
		return err
	}

	coll := conn.GetDatabase().Collection(m.GetCollectionName())
	m.coll = coll

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

func (m BooksModel) Insert(book objects.Book) error {

	_, err := m.coll.InsertOne(context.Background(), book, &options.InsertOneOptions{})

	switch true {
	case mongo.IsDuplicateKeyError(err):
		return errors.DuplicatedObjectIDError.New(book.BookID)
	}

	return err
}

func (m BooksModel) GetByID(bookID string) (objects.Book, error) {

	result := m.coll.FindOne(context.Background(), bson.D{{Key: "book_id", Value: bookID}})

	var book objects.Book
	err := result.Decode(&book)
	if err == nil {
		return book, nil
	}

	switch err {

	case mongo.ErrNoDocuments:
		err = errors.ObjectIDNotFoundError.New(bookID)
	}

	return book, err
}

func (m BooksModel) Search(opt SearchOption) (b models.PaginationData[objects.Book], paginateErr error) {

	if opt.CurrentPage < 1 {
		paginateErr = errors.CurrentPageInvalidError.New()
		return
	}

	matchConditions := bson.A{}
	if !opt.Title.IsNil() {

		matchBson, err := models.CreateMatchBson("title", opt.Title.Value, opt.Title.MatchType)
		if err != nil {
			paginateErr = err
			return
		}

		matchConditions = append(matchConditions, matchBson)
	}

	if !opt.Author.IsNil() {

		matchBson, err := models.CreateMatchBson("author", opt.Author.Value, opt.Author.MatchType)
		if err != nil {
			paginateErr = err
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
		bson.D{{Key: "$skip", Value: (opt.CurrentPage - 1) * m.limit}},
		bson.D{{Key: "$limit", Value: m.limit}},
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

	option := options.Aggregate()

	//NOTE: Research more before applying it
	//option.SetHint("book_id_1")
	pipeline := mongo.Pipeline{matchStage, facetStage, projectStage}

	cur, err := m.coll.Aggregate(context.Background(), pipeline, option)
	if err != nil {
		paginateErr = err
		return
	}

	var aggResultList []models.AggregatedResult[objects.Book]
	err = cur.All(context.Background(), &aggResultList)
	if err != nil {
		paginateErr = err
		return
	}

	aggResult := aggResultList[0]

	totalPages := aggResult.Total / m.limit
	if aggResult.Total%m.limit > 0 {
		totalPages++
	}

	b = models.PaginationData[objects.Book]{
		Data:       aggResult.Data,
		Page:       opt.CurrentPage,
		TotalPages: totalPages,
	}

	return
}

func (m BooksModel) Update(book objects.Book) error {

	filter, err := models.CreateMatchBson("book_id", book.BookID, models.EqualMatchType)
	if err != nil {
		return err
	}

	//TODO: Research to choose update or replace
	result := m.coll.FindOneAndReplace(context.Background(), filter, book)
	if err := result.Err(); err != nil {

		switch err {

		case mongo.ErrNoDocuments:
			return errors.ObjectIDNotFoundError.New(book.BookID)

		default:
			return err
		}
	}

	return nil
}

func (m BooksModel) Delete(bookID string) error {

	filter := bson.D{{Key: "book_id", Value: bookID}}

	option := options.FindOneAndDelete()
	result := m.coll.FindOneAndDelete(context.Background(), filter, option)
	if err := result.Err(); err != nil {

		switch err {

		case mongo.ErrNoDocuments:
			return errors.ObjectIDNotFoundError.New(bookID)

		default:
			return err
		}
	}

	return nil
}
