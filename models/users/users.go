package users

import (
	"context"
	"errors"
	"fmt"
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
	UserID      string              `json:"user_id,omitempty"`
	Username    models.MatchOptions `json:"username,omitempty"`
	AccountName models.MatchOptions `json:"account_name,omitempty"`
	Email       models.MatchOptions `json:"email,omitempty"`
}

type UsersModel struct {
	models.BaseModel[objects.User]
}

const userIDIndex = "user_id_1"
const accountNameIndex = "account_name_1"

func NewUsersModel(conn *mongodb.MongoDBConn, paginateSize ...int) (*UsersModel, error) {

	var searchSize int = 10
	var paginateSizeLen = len(paginateSize)
	if paginateSizeLen > 1 {
		return nil, errors.New("PaginateSize can have only one elements")
	} else if paginateSizeLen == 1 {
		searchSize = paginateSize[0]
	}

	var model = new(UsersModel)

	coll, err := model.createCollection(conn)
	if err != nil {
		return nil, err
	}

	err = model.createIndexes(coll)
	if err != nil {
		return nil, err
	}

	err = model.Inject(coll, searchSize, "user_id")
	if err != nil {
		return nil, err
	}

	return model, nil
}

func (UsersModel) GetCollectionName() string {
	return "users"
}

func (m UsersModel) createCollection(conn *mongodb.MongoDBConn) (*mongo.Collection, error) {

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
				"required": []string{"user_id", "username", "password", "account_name", "email"},
				"properties": bson.M{
					"user_id": bson.M{
						"bsonType":    "string",
						"minimum":     1,
						"description": "User ID must not be empty",
					},
					"username": bson.M{
						"bsonType":    "string",
						"description": "Username must not be empty",
					},
					"password": bson.M{
						"bsonType":    "string",
						"description": "Password must not be empty",
					},
					"account_name": bson.M{
						"bsonType":    "string",
						"minimum":     1,
						"description": "Account name must not be empty",
					},
					"email": bson.M{
						"bsonType":    "string",
						"description": "Email must not be empty",
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

		options := options.RunCmd()
		result := crudDB.RunCommand(context.Background(), cmd, options)
		if err := result.Err(); err != nil {
			return nil, err
		}

		return conn.GetCollection(collectionName), nil
	}

	collectionOptions := options.CreateCollection()
	collectionOptions.SetValidator(validator)
	collectionOptions.SetValidationLevel("strict")

	err = crudDB.CreateCollection(context.Background(), collectionName, collectionOptions)
	if err != nil {
		return nil, err
	}

	return conn.GetCollection(collectionName), nil
}

func (m UsersModel) createIndexes(coll *mongo.Collection) error {

	cur, err := coll.Indexes().List(context.Background())
	if err != nil {
		return err
	}

	var indexes []bson.M
	err = cur.All(context.Background(), &indexes)
	if err != nil {
		return err
	}

	contains := slices.ContainsFunc(indexes, func(m primitive.M) bool {
		return m["name"] == userIDIndex
	})

	if !contains {

		indexModelOptions := options.Index().SetName(userIDIndex).SetUnique(true)
		indexModel := mongo.IndexModel{
			Keys: bson.D{
				{Key: "user_id", Value: 1},
			},
			Options: indexModelOptions,
		}

		_, err = coll.Indexes().CreateOne(context.Background(), indexModel)
		if err != nil {
			return err
		}
	}

	contains = slices.ContainsFunc(indexes, func(m primitive.M) bool {
		return m["name"] == accountNameIndex
	})

	if !contains {

		indexModelOptions := options.Index().SetName(accountNameIndex).SetUnique(true)
		indexModel := mongo.IndexModel{
			Keys: bson.D{
				{Key: "account_name", Value: 1},
			},
			Options: indexModelOptions,
		}

		_, err = coll.Indexes().CreateOne(context.Background(), indexModel)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m UsersModel) Insert(user objects.User) error {

	if strings.EqualFold(user.UserID, "") ||
		strings.EqualFold(user.Username, "") ||
		strings.EqualFold(user.Password, "") ||
		strings.EqualFold(user.AccountName, "") ||
		strings.EqualFold(user.Email, "") {

		//TODO: Add validation failed
		return errors.New(";w;")
	}

	filter := bson.D{
		{
			Key: "$or", Value: bson.A{
				bson.D{{Key: m.ItemIDKey, Value: user.UserID}},
				bson.D{{Key: "username", Value: user.Username}},
				bson.D{{Key: "account_name", Value: user.AccountName}},
				bson.D{{Key: "email", Value: user.Email}},
			},
		},
	}

	err := m.Coll.FindOne(context.Background(), filter).Err()
	if err == nil {
		return serverError.DataAlreadyInUsedError.New()
	}

	if !errors.Is(err, mongo.ErrNoDocuments) {
		return err
	}

	return m.BaseModel.Insert(user)
}

func (m UsersModel) Search(opt SearchOptions) (paginationResult models.PaginationData[objects.User], paginationErr error) {

	var builder = models.NewSearchPipelineBuilder()
	builder.Skip((opt.CurrentPage - 1) * m.BaseModel.SearchLenLimit)
	builder.Limit(m.BaseModel.SearchLenLimit)
	builder.SortedBy([]models.SortData{
		{
			Key:    m.ItemIDKey,
			SortBy: models.SortASC,
		},
	})

	if !strings.EqualFold(opt.UserID, "") {

		fmt.Println("Set user_id")

		paginationErr = builder.Match("user_id", opt.UserID, models.EqualMatchType)
		if paginationErr != nil {
			return
		}
	}

	if !opt.AccountName.IsNil() {

		fmt.Println("Set account_name")

		paginationErr = builder.Match("account_name", opt.AccountName.Value, opt.AccountName.MatchType)
		if paginationErr != nil {
			return
		}
	}

	if !opt.Username.IsNil() {

		fmt.Println("Set username")

		paginationErr = builder.Match("username", opt.Username.Value, opt.Username.MatchType)
		if paginationErr != nil {
			return
		}
	}

	if !opt.Email.IsNil() {

		fmt.Println("Set email")

		paginationErr = builder.Match("email", opt.Email.Value, opt.Email.MatchType)
		if paginationErr != nil {
			return
		}
	}

	pipeline := builder.BuildPipeline()

	paginationResult, paginationErr = m.BaseModel.Search(models.BaseSearchOptions{
		CurrentPage: opt.CurrentPage,
		Pipeline:    pipeline,
	})

	return
}
