package users

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/suite"
	"github.com/supakorn-kn/go-crud/errors"
	"github.com/supakorn-kn/go-crud/mongodb"
	"github.com/supakorn-kn/go-crud/objects"
	"go.mongodb.org/mongo-driver/bson"
)

type UsersModelTestSuite struct {
	suite.Suite
	conn         *mongodb.MongoDBConn
	model        *UsersModel
	insertedUser objects.User
}

func (s *UsersModelTestSuite) SetupSuite() {

	//TODO: Will set parameter from env
	conn := mongodb.New("mongodb://localhost:27017", "go-crud_test")
	s.Require().NoError(conn.Connect(), "Create Mongodb connection failed")

	newModel, err := NewUsersModel(&conn)
	if err != nil {
		conn.Disconnect()
		s.FailNow("Setup User model failed", err)
	}

	s.model = newModel
	s.conn = &conn
}

func (s *UsersModelTestSuite) BeforeTest(suiteName, testName string) {

	if testName == "TestInsert" || testName == "TestSearch" {
		return
	}

	s.insertedUser = fakeUser()
	s.Require().NoError(s.model.Insert(s.insertedUser), "Setup test failed from inserting users")
}

func (s *UsersModelTestSuite) AfterTest(suiteName, testName string) {

	if testName == "TestInsert" || testName == "TestSearch" || testName == "TestDelete" {
		return
	}

	s.Require().NoError(s.model.Delete(s.insertedUser.UserID), "Clearing test failed from deleting users")
}

func (s *UsersModelTestSuite) TearDownSuite() {
	s.conn.Disconnect()
}

func (s *UsersModelTestSuite) TestInsert() {

	s.Run("Should insert valid user properly", func() {

		user := fakeUser()
		s.Require().NoError(s.model.Insert(user), "Inserting User failed")

		result := s.model.Coll.FindOne(context.Background(), bson.D{{Key: "user_id", Value: user.UserID}})

		var actual objects.User
		s.Require().NoError(result.Decode(&actual), "Unmarshalling inserted User failed")
		s.Require().EqualValues(user, actual, "Read data is not the same as inserted")
	})

	s.Run("Should throw error when insert user with existed data", func() {

		user := fakeUser()
		s.Require().NoError(s.model.Insert(user), "Inserting User failed")

		s.Run("Existed user_id", func() {

			newUser := fakeUser()
			newUser.UserID = user.UserID
			s.Require().Error(s.model.Insert(newUser), "Should have thrown error")
		})

		s.Run("Existed username", func() {

			newUser := fakeUser()
			newUser.Username = user.Username
			s.Require().Error(s.model.Insert(newUser), "Should have thrown error")
		})

		s.Run("Existed account_name", func() {

			newUser := fakeUser()
			newUser.AccountName = user.AccountName
			s.Require().Error(s.model.Insert(newUser), "Should have thrown error")
		})

		s.Run("Existed email", func() {

			newUser := fakeUser()
			newUser.Email = user.Email
			s.Require().Error(s.model.Insert(newUser), "Should have thrown error")
		})
	})

	s.Run("Should throw error when insert invalid user data", func() {

		user := fakeUser()

		s.Run("Use empty user ID", func() {

			invalidUser := user
			invalidUser.UserID = ""

			s.Require().Error(s.model.Insert(invalidUser), "Should throw error")
		})

		s.Run("Use empty username", func() {

			invalidUser := user
			invalidUser.Username = ""

			s.Require().Error(s.model.Insert(invalidUser), "Should throw error")
		})

		s.Run("Use empty password", func() {

			invalidUser := user
			invalidUser.Password = ""

			s.Require().Error(s.model.Insert(invalidUser), "Should throw error")
		})

		s.Run("Use empty account name", func() {

			invalidUser := user
			invalidUser.AccountName = ""

			s.Require().Error(s.model.Insert(invalidUser), "Should throw error")
		})

		s.Run("Use empty email", func() {

			invalidUser := user
			invalidUser.Email = ""

			s.Require().Error(s.model.Insert(invalidUser), "Should throw error")
		})
	})
}

func (s *UsersModelTestSuite) TestGetByID() {

	s.Run("Should get the user by user_id properly", func() {

		actual, err := s.model.GetByID(s.insertedUser.UserID)
		s.Require().NoError(err, "Getting exist user failed")
		s.Require().EqualValues(s.insertedUser, actual)
	})

	s.Run("Should throw the error when give non-exist user_id", func() {

		itemID := "non-exist_id"

		actual, err := s.model.GetByID(itemID)
		s.Require().Empty(actual)
		s.Require().ErrorIs(errors.ObjectIDNotFoundError.New(itemID), err, "Should throw error")
	})
}

// func (s *UsersModelTestSuite) TestSearch() {

// 	userA := objects.User{
// 		UserID:      gofakeit.UUID(),
// 		Username:    "user_a",
// 		Password:    "passwd_a",
// 		AccountName: "User A",
// 		Email:       "userA@example.com",
// 	}

// 	userB := objects.User{
// 		UserID:      gofakeit.UUID(),
// 		Username:    "user_b",
// 		Password:    "passwd_b",
// 		AccountName: "User B",
// 		Email:       "userB@example.com",
// 	}

// 	userC := objects.User{
// 		UserID:      gofakeit.UUID(),
// 		Username:    "user_c",
// 		Password:    "passwd_c",
// 		AccountName: "User C",
// 		Email:       "userC@example.com",
// 	}

// 	var initialLimit = s.model.SearchLenLimit
// 	s.model.SearchLenLimit = 2
// 	s.T().Cleanup(func() {

// 		s.model.SearchLenLimit = initialLimit

// 		matchQuery := bson.D{{
// 			Key:   s.model.ItemIDKey,
// 			Value: bson.D{{Key: "$in", Value: bson.A{userA.UserID, userB.UserID, userC.UserID}}},
// 		}}

// 		_, err := s.model.Coll.DeleteMany(context.Background(), matchQuery)
// 		s.NoError(err, "Clearing inserted users for searching failed")
// 	})

// 	users := []objects.User{userA, userB, userC}
// 	for _, user := range users {
// 		s.Require().NoError(s.model.Insert(user), "Insert users before testing failed")
// 	}

// 	slices.SortFunc(users, func(a, b objects.User) int {

// 		if a.UserID > b.UserID {
// 			return 1
// 		}

// 		return -1
// 	})

// 	s.Run("Should get user(s) properly by given options", func() {

// 		var testCases = map[string]struct {
// 			Expected models.PaginationData[objects.User]
// 			Option   SearchOptions
// 		}{
// 			"None (Page 1)": {
// 				Expected: models.PaginationData[objects.User]{
// 					Page:       1,
// 					TotalPages: 2,
// 					Data:       users[:2],
// 				},
// 				Option: SearchOptions{
// 					CurrentPage: 1,
// 				},
// 			},
// 			"None (Page 2)": {
// 				Expected: models.PaginationData[objects.User]{
// 					Page:       2,
// 					TotalPages: 2,
// 					Data:       users[2:],
// 				},
// 				Option: SearchOptions{
// 					CurrentPage: 2,
// 				},
// 			},
// 			"User ID (Equal)": {
// 				Expected: models.PaginationData[objects.User]{
// 					Page:       1,
// 					TotalPages: 1,
// 					Data:       []objects.User{userA},
// 				},
// 				Option: SearchOptions{
// 					CurrentPage: 1,
// 					UserID:      userA.UserID,
// 				},
// 			},
// 			"Username (Equal)": {
// 				Expected: models.PaginationData[objects.User]{
// 					Page:       1,
// 					TotalPages: 1,
// 					Data:       []objects.User{userB},
// 				},
// 				Option: SearchOptions{
// 					CurrentPage: 1,
// 					Username: models.MatchOptions{
// 						MatchType: models.EqualMatchType,
// 						Value:     userB.Username,
// 					},
// 				},
// 			},
// 			"Username (Partial)": {
// 				Expected: models.PaginationData[objects.User]{
// 					Page:       1,
// 					TotalPages: 2,
// 					Data:       users[:2],
// 				},
// 				Option: SearchOptions{
// 					CurrentPage: 1,
// 					Username: models.MatchOptions{
// 						MatchType: models.PartialMatchType,
// 						Value:     "er",
// 					},
// 				},
// 			},
// 			"Username (Start with)": {
// 				Expected: models.PaginationData[objects.User]{
// 					Page:       1,
// 					TotalPages: 2,
// 					Data:       users[:2],
// 				},
// 				Option: SearchOptions{
// 					CurrentPage: 1,
// 					Username: models.MatchOptions{
// 						MatchType: models.StartWithMatchType,
// 						Value:     "us",
// 					},
// 				},
// 			},
// 			"Username (End with)": {
// 				Expected: models.PaginationData[objects.User]{
// 					Page:       1,
// 					TotalPages: 1,
// 					Data:       []objects.User{userC},
// 				},
// 				Option: SearchOptions{
// 					CurrentPage: 1,
// 					Username: models.MatchOptions{
// 						MatchType: models.EndWithMatchType,
// 						Value:     "c",
// 					},
// 				},
// 			},
// 			"Account name (Equal)": {
// 				Expected: models.PaginationData[objects.User]{
// 					Page:       1,
// 					TotalPages: 1,
// 					Data:       []objects.User{userB},
// 				},
// 				Option: SearchOptions{
// 					CurrentPage: 1,
// 					AccountName: models.MatchOptions{
// 						MatchType: models.EqualMatchType,
// 						Value:     userB.AccountName,
// 					},
// 				},
// 			},
// 			"Account name (Partial)": {
// 				Expected: models.PaginationData[objects.User]{
// 					Page:       1,
// 					TotalPages: 2,
// 					Data:       users[:2],
// 				},
// 				Option: SearchOptions{
// 					CurrentPage: 1,
// 					AccountName: models.MatchOptions{
// 						MatchType: models.PartialMatchType,
// 						Value:     "user",
// 					},
// 				},
// 			},
// 			"Account name (Start with)": {
// 				Expected: models.PaginationData[objects.User]{
// 					Page:       1,
// 					TotalPages: 2,
// 					Data:       users[:2],
// 				},
// 				Option: SearchOptions{
// 					CurrentPage: 1,
// 					AccountName: models.MatchOptions{
// 						MatchType: models.StartWithMatchType,
// 						Value:     "use",
// 					},
// 				},
// 			},
// 			"Account name (End with)": {
// 				Expected: models.PaginationData[objects.User]{
// 					Page:       1,
// 					TotalPages: 1,
// 					Data:       []objects.User{userA},
// 				},
// 				Option: SearchOptions{
// 					CurrentPage: 1,
// 					AccountName: models.MatchOptions{
// 						MatchType: models.EndWithMatchType,
// 						Value:     "a",
// 					},
// 				},
// 			},
// 			"Email (Equal)": {
// 				Expected: models.PaginationData[objects.User]{
// 					Page:       1,
// 					TotalPages: 1,
// 					Data:       []objects.User{userB},
// 				},
// 				Option: SearchOptions{
// 					CurrentPage: 1,
// 					Email: models.MatchOptions{
// 						MatchType: models.EqualMatchType,
// 						Value:     userB.Email,
// 					},
// 				},
// 			},
// 			"Email (Partial)": {
// 				Expected: models.PaginationData[objects.User]{
// 					Page:       1,
// 					TotalPages: 2,
// 					Data:       users[:2],
// 				},
// 				Option: SearchOptions{
// 					CurrentPage: 1,
// 					Email: models.MatchOptions{
// 						MatchType: models.PartialMatchType,
// 						Value:     "@",
// 					},
// 				},
// 			},
// 			"Email (Start with)": {
// 				Expected: models.PaginationData[objects.User]{
// 					Page:       1,
// 					TotalPages: 2,
// 					Data:       users[:2],
// 				},
// 				Option: SearchOptions{
// 					CurrentPage: 1,
// 					Email: models.MatchOptions{
// 						MatchType: models.StartWithMatchType,
// 						Value:     "use",
// 					},
// 				},
// 			},
// 			"Email (End with)": {
// 				Expected: models.PaginationData[objects.User]{
// 					Page:       1,
// 					TotalPages: 2,
// 					Data:       users[:2],
// 				},
// 				Option: SearchOptions{
// 					CurrentPage: 1,
// 					Email: models.MatchOptions{
// 						MatchType: models.EndWithMatchType,
// 						Value:     "m",
// 					},
// 				},
// 			},
// 		}

// 		for optionName, testCase := range testCases {

// 			s.Run(fmt.Sprintf("Search with option %s", optionName), func() {

// 				fmt.Println(optionName)

// 				paginationData, err := s.model.Search(testCase.Option)
// 				s.Require().NoError(err, "Searching user failed")
// 				s.Require().EqualValues(testCase.Expected, paginationData)
// 			})
// 		}
// 	})

// 	s.Run("Should throw error when set current page as non-positive value", func() {

// 		result, err := s.model.Search(SearchOptions{CurrentPage: 0})
// 		s.Require().ErrorIs(errors.CurrentPageInvalidError.New(), err, "Should have returned error")
// 		s.Require().Empty(result)
// 	})

// 	s.Run("Should throw error when set invalid or unsupported match type", func() {

// 		result, err := s.model.Search(
// 			SearchOptions{
// 				CurrentPage: 1,
// 				Username: models.MatchOptions{
// 					MatchType: 255,
// 				},
// 			},
// 		)
// 		s.Require().Error(err, "Should have returned error")
// 		s.Require().Empty(result)
// 	})

// }

func (s *UsersModelTestSuite) TestUpdate() {

	s.Run("Should update exist user properly", func() {

		userToUpdate := fakeUser()
		userToUpdate.UserID = s.insertedUser.UserID

		s.Require().NoError(s.model.Update(userToUpdate))

		s.T().Cleanup(func() {
			s.Require().NoError(s.model.Update(s.insertedUser))
		})

		actual, err := s.model.GetByID(userToUpdate.UserID)
		s.Require().NoError(err, "Getting updated user failed")
		s.Require().EqualValues(userToUpdate, actual)
	})

	s.Run("Should update partial data in user properly", func() {

		mockUser := fakeUser()

		var userToUpdate = objects.User{
			UserID:   s.insertedUser.UserID,
			Password: mockUser.Password,
			Email:    mockUser.Email,
		}

		s.Require().NoError(s.model.Update(userToUpdate))

		s.T().Cleanup(func() {
			s.Require().NoError(s.model.Update(s.insertedUser))
		})

		expected := s.insertedUser
		expected.Password = userToUpdate.Password
		expected.Email = userToUpdate.Email

		actual, err := s.model.GetByID(expected.UserID)
		s.Require().NoError(err, "Getting updated user failed")
		s.Require().EqualValues(expected, actual)
	})

	s.Run("Should throw error when update non-exist user", func() {

		userToUpdate := fakeUser()
		s.Require().Error(s.model.Update(userToUpdate))

		actual, err := s.model.GetByID(s.insertedUser.UserID)
		s.Require().NoError(err, "Getting updated user failed")
		s.Require().EqualValues(s.insertedUser, actual)
	})
}

func (s *UsersModelTestSuite) TestDelete() {

	s.Run("Should delete exist user properly", func() {

		s.Require().NoError(s.model.Delete(s.insertedUser.UserID))

		actual, err := s.model.GetByID(s.insertedUser.UserID)
		s.Require().Error(err, "Should throw error after getting deleted user")
		s.Require().Empty(actual, "The user should have been empty")
	})

	s.Run("Should throw error when delete non-exist user", func() {

		s.Require().Error(s.model.Delete("invalid_user_id"), "Delete exist user failed")
	})
}

func TestUsersModel(t *testing.T) {

	suite.Run(t, new(UsersModelTestSuite))
}

func fakeUser() objects.User {

	now := time.Now().UnixMilli()

	return objects.User{
		UserID:      gofakeit.UUID(),
		Username:    fmt.Sprintf("username_%d", now),
		Password:    fmt.Sprintf("password_%d", now),
		AccountName: fmt.Sprintf("acct_%d", now),
		Email:       fmt.Sprintf("mail_%d@example.com", now),
	}
}
