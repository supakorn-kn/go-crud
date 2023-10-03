package models

import (
	"fmt"
	"reflect"

	crud_errors "github.com/supakorn-kn/go-crud/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AggregatedResult[T any] struct {
	Count int `bson:"count"`
	Total int `bson:"total"`
	Data  []T `bson:"data"`
}

type MatchType uint8

func (enum MatchType) String() string {
	return [...]string{"Equal", "Partial", "Start with", "End with", "Contains in"}[enum]
}

const (
	EqualMatchType      = 0
	PartialMatchType    = 1
	StartWithMatchType  = 2
	EndWithMatchType    = 3
	ContainsInMatchType = 4
)

type BaseSearchOption struct {
	CurrentPage int
	Pipeline    mongo.Pipeline
}

type MatchOption struct {
	MatchType MatchType `json:"match_type"`
	Value     string    `json:"value"`
}

func (opt MatchOption) IsNil() bool {
	return reflect.ValueOf(opt).IsZero()
}

type SortOrder = int

const (
	SortASC  SortOrder = 1
	SortDESC SortOrder = -1
)

type SortData struct {
	Key    string
	SortBy SortOrder
}

type SearchPipelineBuilder struct {
	matches bson.A
	sorts   bson.D
	skip    bson.D
	limit   bson.D

	sortKeys  map[string]struct{}
	matchKeys map[string]struct{}
}

func NewSearchPipelineBuilder() *SearchPipelineBuilder {
	return new(SearchPipelineBuilder)
}

func (b *SearchPipelineBuilder) SortedBy(sortDataList []SortData) error {

	if len(sortDataList) == 0 {

		return crud_errors.SortListInvalidError.New()
	}

	if b.sorts == nil {
		b.sorts = bson.D{{Key: "$sort", Value: bson.D{}}}
		b.sortKeys = make(map[string]struct{}, 0)
	}

	var sortList = b.sorts[0].Value.(bson.D)
	for _, sortData := range sortDataList {

		if _, found := b.sortKeys[sortData.Key]; found {

			return crud_errors.MatchKeyDuplicatedError.New(sortData.Key)
		}

		sortList = append(sortList, bson.E{Key: sortData.Key, Value: sortData.SortBy})
		b.sortKeys[sortData.Key] = struct{}{}
	}

	b.sorts[0].Value = sortList

	return nil
}

func (b *SearchPipelineBuilder) Match(key string, value any, matchType MatchType) error {

	if b.matches == nil {
		b.matches = bson.A{}
		b.matchKeys = make(map[string]struct{}, 0)
	}

	if _, found := b.matchKeys[key]; found {

		return crud_errors.MatchKeyDuplicatedError.New(key)
	}

	var matchQuery, err = CreateMatchBson(key, value, matchType)
	if err != nil {
		return err
	}

	b.matches = append(b.matches, matchQuery)
	b.matchKeys[key] = struct{}{}
	return nil
}

func (b *SearchPipelineBuilder) Skip(count int) {
	b.skip = bson.D{{Key: "$skip", Value: count}}
}

func (b *SearchPipelineBuilder) Limit(count int) {
	b.limit = bson.D{{Key: "$limit", Value: count}}
}

func (b *SearchPipelineBuilder) BuildPipeline() mongo.Pipeline {

	var matchStage = b.createMatchStage()
	var facetStage = b.createFacetStage()
	var projectStage = b.createProjectStage()

	return mongo.Pipeline{matchStage, facetStage, projectStage}
}

func (b *SearchPipelineBuilder) createMatchStage() bson.D {

	if len(b.matches) == 0 {
		b.matches = append(b.matches, bson.D{})
	}

	return bson.D{
		{
			Key: "$match", Value: bson.D{
				{Key: "$and", Value: b.matches},
			},
		},
	}
}

func (b *SearchPipelineBuilder) createFacetStage() bson.D {

	var paginateResultQuery = bson.A{}

	if b.sorts != nil {
		paginateResultQuery = append(paginateResultQuery, b.sorts)
	}

	if b.skip != nil {
		paginateResultQuery = append(paginateResultQuery, b.skip)
	}

	if b.limit != nil {
		paginateResultQuery = append(paginateResultQuery, b.limit)
	}

	var groupQuery = bson.D{{Key: "$group", Value: bson.D{
		{Key: "_id", Value: nil},
		{Key: "data", Value: bson.D{{Key: "$push", Value: "$$ROOT"}}},
		{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
	}}}

	paginateResultQuery = append(paginateResultQuery, groupQuery)

	matchResultQuery := bson.A{
		bson.D{{Key: "$count", Value: "total"}},
	}

	return bson.D{
		{
			Key: "$facet", Value: bson.D{
				{Key: "paginate_result", Value: paginateResultQuery},
				{Key: "match_result", Value: matchResultQuery},
			},
		},
	}
}

func (b *SearchPipelineBuilder) createProjectStage() bson.D {

	return bson.D{
		{
			Key: "$project", Value: bson.D{
				{Key: "count", Value: bson.D{{Key: "$first", Value: "$paginate_result.count"}}},
				{Key: "total", Value: bson.D{{Key: "$first", Value: "$match_result.total"}}},
				{Key: "data", Value: bson.D{{Key: "$first", Value: "$paginate_result.data"}}},
			},
		},
	}
}

func CreateMatchBson(key string, value any, matchType MatchType) (bson.D, error) {

	switch matchType {

	case EqualMatchType:
		return EqualMatchBson(key, value), nil

	case PartialMatchType:
		return PartialMatchBson(key, value), nil

	case StartWithMatchType:
		return StartWithMatchBson(key, value), nil

	case EndWithMatchType:
		return EndWithMatchBson(key, value), nil

	case ContainsInMatchType:
		return ContainsInMatchBson(key, value)

	default:
		return nil, crud_errors.MatchTypeInvalidError.New(matchType)
	}
}

// EqualMatchBson creates BSON for equal search (Case-sensitive)
func EqualMatchBson(key string, value any) bson.D {
	return bson.D{{Key: key, Value: value}}
}

// PartialMatchBson creates BSON for partial search (Case-insensitive)
func PartialMatchBson(key string, value any) bson.D {
	return bson.D{{Key: key, Value: bson.M{"$regex": value, "$options": "i"}}}
}

// StartWithMatchBson creates BSON for start with keyword search (Case-insensitive)
func StartWithMatchBson(key string, value any) bson.D {
	format := fmt.Sprintf("^%s", value)
	return bson.D{{Key: key, Value: bson.M{"$regex": format, "$options": "im"}}}
}

// EndWithMatchBson creates BSON for end with keyword search (Case-insensitive)
func EndWithMatchBson(key string, value any) bson.D {
	format := fmt.Sprintf("%s$", value)
	return bson.D{{Key: key, Value: bson.M{"$regex": format, "$options": "im"}}}
}

// EndWithMatchBson creates BSON for end with keyword search (Case-insensitive)
func ContainsInMatchBson(key string, value any) (bson.D, error) {

	var valueType = reflect.TypeOf(value).Kind()
	if valueType != reflect.Array && valueType != reflect.Slice {

		return nil, crud_errors.MatchValueInvalidError.New(value, ContainsInMatchType)
	}

	return bson.D{{Key: key, Value: bson.D{{Key: "$in", Value: value}}}}, nil
}
