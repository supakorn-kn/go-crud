package models

import (
	"fmt"
	"reflect"

	"github.com/supakorn-kn/go-crud/errors"
	"go.mongodb.org/mongo-driver/bson"
)

type AggregatedResult[T any] struct {
	Count int `bson:"count"`
	Total int `bson:"total"`
	Data  []T `bson:"data"`
}

type MatchType uint8

const (
	EqualMatchType = iota
	PartialMatchType
	StartWithMatchType
	EndWithMatchType
)

type MatchOption struct {
	MatchType MatchType `json:"match_type"`
	Value     string    `json:"value"`
}

func (opt MatchOption) IsNil() bool {
	return reflect.ValueOf(opt).IsZero()
}

// TODO: Will change value type
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

	default:
		return nil, errors.MatchTypeInvalidError.New(matchType)
	}
}

// EqualMatchBson creates BSON for equal search (Case-sensitive)
// TODO: Will change value type
func EqualMatchBson(key string, value any) bson.D {
	return bson.D{{Key: key, Value: value}}
}

// PartialMatchBson creates BSON for partial search (Case-insensitive)
// TODO: Will change value type
func PartialMatchBson(key string, value any) bson.D {
	return bson.D{{Key: key, Value: bson.M{"$regex": value, "$options": "i"}}}
}

// StartWithMatchBson creates BSON for start with keyword search (Case-insensitive)
// TODO: Will change value type
func StartWithMatchBson(key string, value any) bson.D {
	format := fmt.Sprintf("^%s", value)
	return bson.D{{Key: key, Value: bson.M{"$regex": format, "$options": "im"}}}
}

// EndWithMatchBson creates BSON for end with keyword search (Case-insensitive)
// TODO: Will change value type
func EndWithMatchBson(key string, value any) bson.D {
	format := fmt.Sprintf("%s$", value)
	return bson.D{{Key: key, Value: bson.M{"$regex": format, "$options": "im"}}}
}