package models

import (
	"context"
	"errors"

	serverError "github.com/supakorn-kn/go-crud/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Item interface {
	GetID() string
}

type PaginationData[Data Item] struct {
	Page       int    `json:"page"`
	TotalPages int    `json:"total_pages"`
	Data       []Data `json:"data"`
}

type Model[T Item] interface {
	BaseModel[Item]

	GetCollectionName() string
	Insert(item Item) error
	GetByID(itemID string) (Item, error)
	Search() (PaginationData[Item], error)
	Update(item Item) error
	Delete(itemID string) error
}

type BaseModel[item Item] struct {
	SearchLenLimit int

	Coll      *mongo.Collection
	ItemIDKey string
}

func (m *BaseModel[T]) Inject(conn *mongo.Collection, searchLenLimit int, itemIDKey string) error {

	if searchLenLimit < 1 {
		return errors.New("PaginateSize value can be only positive integer")
	}

	m.Coll = conn
	m.SearchLenLimit = searchLenLimit
	m.ItemIDKey = itemIDKey

	return nil
}

func (m BaseModel[T]) Insert(item T) error {

	_, err := m.Coll.InsertOne(context.TODO(), item)
	if err != nil {

		if mongo.IsDuplicateKeyError(err) {
			return serverError.DuplicatedObjectIDError.New(item.GetID())
		}

		return err
	}

	return nil
}

func (m BaseModel[T]) GetByID(itemID string) (item T, err error) {

	result := m.Coll.FindOne(context.TODO(), bson.D{{Key: m.ItemIDKey, Value: itemID}})

	err = result.Decode(&item)
	if errors.Is(err, mongo.ErrNoDocuments) {
		err = serverError.ObjectIDNotFoundError.New(itemID)
		return
	}

	return
}

func (m BaseModel[T]) Search(opt BaseSearchOption) (paginationData PaginationData[T], paginateErr error) {

	var currentPage = opt.CurrentPage
	if currentPage < 1 {
		paginateErr = serverError.CurrentPageInvalidError.New()
		return
	}

	var cur *mongo.Cursor
	cur, paginateErr = m.Coll.Aggregate(context.TODO(), opt.Pipeline)
	if paginateErr != nil {
		return
	}

	var aggResultList []AggregatedResult[T]
	paginateErr = cur.All(context.TODO(), &aggResultList)
	if paginateErr != nil {
		return
	}

	aggResult := aggResultList[0]

	totalPages := aggResult.Total / m.SearchLenLimit
	if aggResult.Total%m.SearchLenLimit > 0 {
		totalPages++
	}

	paginationData = PaginationData[T]{
		Data:       aggResult.Data,
		Page:       currentPage,
		TotalPages: totalPages,
	}

	return
}

func (m BaseModel[T]) Update(item T) error {

	filter, err := CreateMatchBson(m.ItemIDKey, item.GetID(), EqualMatchType)
	if err != nil {
		return err
	}

	b, err := bson.Marshal(item)
	if err != nil {
		return err
	}

	var parsedBson bson.D
	err = bson.Unmarshal(b, &parsedBson)
	if err != nil {
		return err
	}

	var updateBson bson.D
	for _, keyValue := range parsedBson {

		if keyValue.Key != m.ItemIDKey {
			updateBson = append(updateBson, keyValue)
		}
	}

	result := m.Coll.FindOneAndUpdate(context.TODO(), filter, bson.D{{Key: "$set", Value: updateBson}})
	if err := result.Err(); err != nil {

		if errors.Is(err, mongo.ErrNoDocuments) {
			return serverError.ObjectIDNotFoundError.New(item.GetID())
		}

		return err
	}

	return nil
}

func (m BaseModel[T]) Delete(itemID string) error {

	filter := bson.D{{Key: m.ItemIDKey, Value: itemID}}

	result := m.Coll.FindOneAndDelete(context.TODO(), filter)
	if err := result.Err(); err != nil {

		if errors.Is(err, mongo.ErrNoDocuments) {
			return serverError.ObjectIDNotFoundError.New(itemID)
		}

		return err
	}

	return nil
}
