package models

import (
	"context"
	"fmt"

	"github.com/supakorn-kn/go-crud/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Item interface {
	GetID() string
}

type SearchOption struct {
	CurrentPage int
	Pipeline    mongo.Pipeline
}

type BaseModel[item Item] struct {
	SearchLenLimit int

	Coll      *mongo.Collection
	ItemIDKey string
}

func NewBaseModel[item Item](conn *mongo.Collection, searchLenLimit int, itemIDKey string) (*BaseModel[item], error) {

	if searchLenLimit < 1 {
		return nil, fmt.Errorf("PaginateSize value can be only positive integer")
	}

	m := BaseModel[item]{
		Coll:           conn,
		SearchLenLimit: searchLenLimit,
		ItemIDKey:      itemIDKey,
	}

	return &m, nil
}

func (m BaseModel[T]) Insert(item T) error {

	_, err := m.Coll.InsertOne(context.Background(), item)
	if err != nil {

		switch true {
		case mongo.IsDuplicateKeyError(err):
			return errors.DuplicatedObjectIDError.New(item.GetID())
		default:
			return err
		}
	}

	return nil
}

func (m BaseModel[T]) GetByID(itemID string) (item T, err error) {

	result := m.Coll.FindOne(context.Background(), bson.D{{Key: m.ItemIDKey, Value: itemID}})

	err = result.Decode(&item)
	switch err {

	case mongo.ErrNoDocuments:
		err = errors.ObjectIDNotFoundError.New(itemID)
	}

	return
}

func (m BaseModel[T]) Search(opt SearchOption) (paginationData PaginationData[T], paginateErr error) {

	var currentPage = opt.CurrentPage

	if currentPage < 1 {
		paginateErr = errors.CurrentPageInvalidError.New()
		return
	}

	cur, err := m.Coll.Aggregate(context.Background(), opt.Pipeline)
	if err != nil {
		paginateErr = err
		return
	}

	var aggResultList []AggregatedResult[T]
	err = cur.All(context.Background(), &aggResultList)
	if err != nil {
		paginateErr = err
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

	//TODO: Research to choose update or replace
	result := m.Coll.FindOneAndReplace(context.Background(), filter, item)
	if err := result.Err(); err != nil {

		switch err {

		case mongo.ErrNoDocuments:
			return errors.ObjectIDNotFoundError.New(item.GetID())

		default:
			return err
		}
	}

	return nil
}

func (m BaseModel[T]) Delete(itemID string) error {

	filter := bson.D{{Key: m.ItemIDKey, Value: itemID}}

	result := m.Coll.FindOneAndDelete(context.Background(), filter)
	if err := result.Err(); err != nil {

		switch err {

		case mongo.ErrNoDocuments:
			return errors.ObjectIDNotFoundError.New(itemID)

		default:
			return err
		}
	}

	return nil
}
