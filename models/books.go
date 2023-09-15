package models

import (
	"context"
	"reflect"
	"slices"

	"github.com/supakorn-kn/go-crud/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Book struct {
	BookID      string   `json:"book_id" bson:"book_id"`
	Title       string   `json:"title" bson:"title"`
	Author      string   `json:"author" bson:"author"`
	Description string   `json:"description" bson:"description"`
	Categories  []string `json:"categories" bson:"categories"`
}

func (b *Book) IsNil() bool {
	return reflect.ValueOf(b).IsZero()
}

type AggregatedResult struct {
	Count int    `bson:"count"`
	Total int    `bson:"total"`
	Data  []Book `bson:"data"`
}

type PaginationData struct {
	Page       int    `json:"page"`
	TotalPages int    `json:"total_pages"`
	Data       []Book `json:"data"`
}

type PaginateOption struct {
	Title       string   `json:"title,omitempty"`
	Author      string   `json:"author,omitempty"`
	Categories  []string `json:"categories,omitempty"`
	CurrentPage int      `json:"current_page"`
}

type BooksModel struct {
	coll *mongo.Collection
}

func (m BooksModel) GetCollectionName() string {
	return "books_info"
}

func (m *BooksModel) Init(conn *mongodb.MongoDBConn) error {

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
	collectionNameList, err := bookDB.ListCollectionNames(context.TODO(), filter, option)
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
		result := bookDB.RunCommand(context.TODO(), cmd, option)
		if err := result.Err(); err != nil {
			return err
		}

		return nil
	}

	collectionOption := options.CreateCollection()
	collectionOption.SetValidator(validator)
	collectionOption.SetValidationLevel("strict")

	err = bookDB.CreateCollection(context.TODO(), collectionName, collectionOption)

	if err != nil {
		return err
	}

	return nil
}

func (m BooksModel) initIndexes(conn *mongodb.MongoDBConn) error {

	collectionName := m.GetCollectionName()
	coll := conn.GetDatabase().Collection(collectionName)
	cur, err := coll.Indexes().List(context.TODO())
	if err != nil {
		return nil
	}

	var indexName = "title_1_author_1"

	var indexes []bson.M
	err = cur.All(context.TODO(), &indexes)
	if err != nil {
		return err
	}

	contains := slices.ContainsFunc(indexes, func(m primitive.M) bool {
		return m["name"] == indexName
	})

	if contains {
		return nil
	}

	indexModelOption := options.Index()
	indexModelOption.SetName(indexName)

	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "title", Value: 1},
			{Key: "author", Value: 1},
		},
		Options: indexModelOption,
	}

	option := options.CreateIndexes()
	_, err = coll.Indexes().CreateOne(context.TODO(), indexModel, option)
	if err != nil {
		return err
	}

	return nil
}

func (m BooksModel) Create(book Book) error {

	_, err := m.coll.InsertOne(context.TODO(), book, &options.InsertOneOptions{})
	return err
}

func (m BooksModel) Paginate(opt PaginateOption) (b PaginationData, paginateErr error) {

	//TODO: Use config variable instead
	var limit = 10
	var skip = (opt.CurrentPage - 1) * limit

	paginateResultQuery := bson.A{
		bson.D{{Key: "$skip", Value: skip}},
		bson.D{{Key: "$limit", Value: limit}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "data", Value: bson.D{{Key: "$push", Value: "$$ROOT"}}},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}
	matchResultQuery := bson.A{
		bson.D{{Key: "$count", Value: "total"}},
	}

	matchConditions := bson.A{}
	if opt.Title != "" {
		cond := bson.D{{Key: "title", Value: bson.D{{Key: "$regex", Value: opt.Title}}}}
		matchConditions = append(matchConditions, cond)
	}

	if opt.Author != "" {
		cond := bson.D{{Key: "author", Value: bson.D{{Key: "$regex", Value: opt.Author}}}}
		matchConditions = append(matchConditions, cond)
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
	pipeline := mongo.Pipeline{matchStage, facetStage, projectStage}

	cur, err := m.coll.Aggregate(context.TODO(), pipeline, option)
	if err != nil {
		paginateErr = err
		return
	}

	var aggResultList []AggregatedResult
	err = cur.All(context.TODO(), &aggResultList)
	if err != nil {
		paginateErr = err
		return
	}

	aggResult := aggResultList[0]

	totalPages := aggResult.Total / limit
	if aggResult.Total%limit > 0 {
		totalPages++
	}

	b = PaginationData{
		Data:       aggResult.Data,
		Page:       opt.CurrentPage,
		TotalPages: totalPages,
	}

	return
}

func (m BooksModel) Update(book Book) error {

	filter := bson.D{{Key: "book_id", Value: book.BookID}}

	//TODO: Research to choose update or replace
	option := options.FindOneAndReplace()
	result := m.coll.FindOneAndReplace(context.TODO(), filter, option)
	if err := result.Err(); err != nil {
		return err
	}

	return nil
}

func (m BooksModel) Delete(bookID string) error {

	filter := bson.D{{Key: "book_id", Value: bookID}}

	option := options.FindOneAndDelete()
	result := m.coll.FindOneAndDelete(context.TODO(), filter, option)
	if err := result.Err(); err != nil {
		return err
	}

	return nil
}
