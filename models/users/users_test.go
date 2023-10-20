package users

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/suite"
	"github.com/supakorn-kn/go-crud/env"
	"github.com/supakorn-kn/go-crud/errors"
	"github.com/supakorn-kn/go-crud/models"
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

	config, err := env.GetEnv()
	s.Require().NoError(err)

	conn, err := mongodb.New(config.MongoDB)
	s.Require().NoError(err, "Create MongoDB connection failed")
	s.Require().NoError(conn.Connect(), "Connecting to MongoDB failed")

	newModel, err := NewUsersModel(conn)
	if err != nil {
		conn.Disconnect()
		s.FailNow("Setup User model failed", err)
	}

	s.model = newModel
	s.conn = conn
}

func (s *UsersModelTestSuite) BeforeTest(suiteName, testName string) {

	if testName == "TestInsert" || testName == "TestSearch" {
		return
	}

	s.insertedUser = mockUser()
	s.Require().NoError(s.model.Insert(s.insertedUser), "Setup test failed from inserting users")
}

func (s *UsersModelTestSuite) AfterTest(suiteName, testName string) {

	if testName == "TestSearch" || testName == "TestDelete" {
		return
	}

	s.Require().NoError(s.model.Delete(s.insertedUser.UserID), "Clearing test failed from deleting users")
}

func (s *UsersModelTestSuite) TearDownSuite() {
	s.conn.Disconnect()
}

func (s *UsersModelTestSuite) TestInsert() {

	s.Run("Should insert valid user properly", func() {

		user := mockUser()
		s.Require().NoError(s.model.Insert(user), "Inserting User failed")

		result := s.model.Coll.FindOne(context.Background(), bson.D{{Key: "user_id", Value: user.UserID}})

		var actual objects.User
		s.Require().NoError(result.Decode(&actual), "Unmarshalling inserted User failed")
		s.Require().EqualValues(user, actual, "Read data is not the same as inserted")

		s.insertedUser = user
	})

	s.Run("Should throw error when insert user with existed data", func() {

		user := mockUser()
		s.Require().NoError(s.model.Insert(user), "Inserting User failed")

		s.T().Cleanup(func() {
			s.Require().NoError(s.model.Delete(user.UserID), "Clearing test failed from deleting user")
		})

		s.Run("Existed user_id", func() {

			newUser := mockUser()
			newUser.UserID = user.UserID
			s.Require().Error(s.model.Insert(newUser), "Should have thrown error")
		})

		s.Run("Existed username", func() {

			newUser := mockUser()
			newUser.Username = user.Username
			s.Require().Error(s.model.Insert(newUser), "Should have thrown error")
		})

		s.Run("Existed account_name", func() {

			newUser := mockUser()
			newUser.AccountName = user.AccountName
			s.Require().Error(s.model.Insert(newUser), "Should have thrown error")
		})

		s.Run("Existed email", func() {

			newUser := mockUser()
			newUser.Email = user.Email
			s.Require().Error(s.model.Insert(newUser), "Should have thrown error")
		})
	})

	s.Run("Should throw error when insert invalid user data", func() {

		user := mockUser()

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

		s.Run("Use invalid email format", func() {

			invalidUser := user
			invalidUser.Email = "invalid-email"

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

func (s *UsersModelTestSuite) TestSearch() {

	userA := objects.User{
		UserID:      gofakeit.UUID(),
		Username:    "user_search_a",
		Password:    "passwd_search_a",
		AccountName: "Search_User A",
		Email:       "mail_search_userA@example.mock",
	}

	userB := objects.User{
		UserID:      gofakeit.UUID(),
		Username:    "user_search_b",
		Password:    "passwd_search_b",
		AccountName: "Search_User B",
		Email:       "mail_search_userB@example.mock",
	}

	userC := objects.User{
		UserID:      gofakeit.UUID(),
		Username:    "user_search_c",
		Password:    "passwd_search_c",
		AccountName: "Search_User C",
		Email:       "mail_search_userC@example.mock",
	}

	users := []objects.User{userA, userB, userC}
	for _, user := range users {
		s.Require().NoError(s.model.Insert(user), "Insert users before testing failed")
	}

	slices.SortFunc(users, func(a, b objects.User) int {

		if a.UserID > b.UserID {
			return 1
		}

		return -1
	})

	var initialLimit = s.model.SearchLenLimit
	s.model.SearchLenLimit = 2

	s.T().Cleanup(func() {

		s.model.SearchLenLimit = initialLimit

		matchQuery := bson.D{{
			Key:   s.model.ItemIDKey,
			Value: bson.D{{Key: "$in", Value: bson.A{userA.UserID, userB.UserID, userC.UserID}}},
		}}

		_, err := s.model.Coll.DeleteMany(context.Background(), matchQuery)
		s.NoError(err, "Clearing inserted users for searching failed")
	})

	count, err := s.model.Coll.CountDocuments(context.Background(), bson.D{})
	var allDocsCount = int(count)

	s.Require().NoError(err)
	totalPages := allDocsCount / s.model.SearchLenLimit
	if allDocsCount%s.model.SearchLenLimit != 0 {
		totalPages++
	}

	s.Run("Should get user(s) properly by given options", func() {

		var testCases = map[string]struct {
			Expected models.PaginationData[objects.User]
			Validate func(*UsersModelTestSuite, models.PaginationData[objects.User])
			Option   SearchOptions
		}{
			"None (Page 1)": {
				Validate: func(s *UsersModelTestSuite, result models.PaginationData[objects.User]) {

					s.Equal(1, result.Page)
					s.Equal(totalPages, result.TotalPages)
					s.Equal(allDocsCount, result.Count)
					s.Len(result.Data, 2)
				},
				Option: SearchOptions{
					CurrentPage: 1,
				},
			},
			"None (Page 2)": {
				Expected: models.PaginationData[objects.User]{
					Page:       2,
					TotalPages: 2,
					Data:       users[2:],
				},
				Validate: func(s *UsersModelTestSuite, result models.PaginationData[objects.User]) {

					s.Equal(2, result.Page)
					s.Equal(totalPages, result.TotalPages)
					s.Equal(allDocsCount, result.Count)

					dataLen := len(result.Data)
					s.GreaterOrEqual(dataLen, 1)
					s.Less(dataLen, 3)
				},
				Option: SearchOptions{
					CurrentPage: 2,
				},
			},
			"User ID": {
				Validate: func(s *UsersModelTestSuite, result models.PaginationData[objects.User]) {

					var expected = models.PaginationData[objects.User]{
						Page:       1,
						TotalPages: 1,
						Count:      1,
						Data:       []objects.User{userA},
					}

					s.Equal(expected, result)
				},
				Option: SearchOptions{
					CurrentPage: 1,
					UserID:      userA.UserID,
				},
			},
			"Username (Equal)": {
				Validate: func(s *UsersModelTestSuite, result models.PaginationData[objects.User]) {

					var expected = models.PaginationData[objects.User]{
						Page:       1,
						TotalPages: 1,
						Count:      1,
						Data:       []objects.User{userB},
					}

					s.Equal(expected, result)
				},
				Option: SearchOptions{
					CurrentPage: 1,
					Username: models.MatchOptions{
						MatchType: models.EqualMatchType,
						Value:     userB.Username,
					},
				},
			},
			"Username (Partial)": {
				Validate: func(s *UsersModelTestSuite, result models.PaginationData[objects.User]) {

					var expected = models.PaginationData[objects.User]{
						Page:       1,
						TotalPages: 2,
						Count:      3,
						Data:       users[:2],
					}

					s.Equal(expected, result)
				},
				Option: SearchOptions{
					CurrentPage: 1,
					Username: models.MatchOptions{
						MatchType: models.PartialMatchType,
						Value:     "_search_",
					},
				},
			},
			"Username (Start with)": {
				Validate: func(s *UsersModelTestSuite, result models.PaginationData[objects.User]) {

					var expected = models.PaginationData[objects.User]{
						Page:       1,
						TotalPages: 2,
						Count:      3,
						Data:       users[:2],
					}

					s.Equal(expected, result)
				},
				Option: SearchOptions{
					CurrentPage: 1,
					Username: models.MatchOptions{
						MatchType: models.StartWithMatchType,
						Value:     "user_",
					},
				},
			},
			"Username (End with)": {
				Validate: func(s *UsersModelTestSuite, result models.PaginationData[objects.User]) {

					var expected = models.PaginationData[objects.User]{
						Page:       1,
						TotalPages: 1,
						Count:      1,
						Data:       []objects.User{userC},
					}

					s.Equal(expected, result)
				},
				Option: SearchOptions{
					CurrentPage: 1,
					Username: models.MatchOptions{
						MatchType: models.EndWithMatchType,
						Value:     "_c",
					},
				},
			},
			"Account name (Equal)": {
				Validate: func(s *UsersModelTestSuite, result models.PaginationData[objects.User]) {

					var expected = models.PaginationData[objects.User]{
						Page:       1,
						TotalPages: 1,
						Count:      1,
						Data:       []objects.User{userB},
					}

					s.Equal(expected, result)
				},
				Option: SearchOptions{
					CurrentPage: 1,
					AccountName: models.MatchOptions{
						MatchType: models.EqualMatchType,
						Value:     userB.AccountName,
					},
				},
			},
			"Account name (Partial)": {
				Validate: func(s *UsersModelTestSuite, result models.PaginationData[objects.User]) {

					var expected = models.PaginationData[objects.User]{
						Page:       1,
						TotalPages: 2,
						Count:      3,
						Data:       users[:2],
					}

					s.Equal(expected, result)
				},
				Option: SearchOptions{
					CurrentPage: 1,
					AccountName: models.MatchOptions{
						MatchType: models.PartialMatchType,
						Value:     "_user",
					},
				},
			},
			"Account name (Start with)": {
				Validate: func(s *UsersModelTestSuite, result models.PaginationData[objects.User]) {

					var expected = models.PaginationData[objects.User]{
						Page:       1,
						TotalPages: 2,
						Count:      3,
						Data:       users[:2],
					}

					s.Equal(expected, result)
				},
				Option: SearchOptions{
					CurrentPage: 1,
					AccountName: models.MatchOptions{
						MatchType: models.StartWithMatchType,
						Value:     "search_",
					},
				},
			},
			"Account name (End with)": {
				Validate: func(s *UsersModelTestSuite, result models.PaginationData[objects.User]) {

					var expected = models.PaginationData[objects.User]{
						Page:       1,
						TotalPages: 1,
						Count:      1,
						Data:       []objects.User{userA},
					}

					s.Equal(expected, result)
				},
				Option: SearchOptions{
					CurrentPage: 1,
					AccountName: models.MatchOptions{
						MatchType: models.EndWithMatchType,
						Value:     "user a",
					},
				},
			},
			"Email (Equal)": {
				Validate: func(s *UsersModelTestSuite, result models.PaginationData[objects.User]) {

					var expected = models.PaginationData[objects.User]{
						Page:       1,
						TotalPages: 1,
						Count:      1,
						Data:       []objects.User{userB},
					}

					s.Equal(expected, result)
				},
				Option: SearchOptions{
					CurrentPage: 1,
					Email: models.MatchOptions{
						MatchType: models.EqualMatchType,
						Value:     userB.Email,
					},
				},
			},
			"Email (Partial)": {
				Validate: func(s *UsersModelTestSuite, result models.PaginationData[objects.User]) {

					var expected = models.PaginationData[objects.User]{
						Page:       1,
						TotalPages: 2,
						Count:      3,
						Data:       users[:2],
					}

					s.Equal(expected, result)
				},
				Option: SearchOptions{
					CurrentPage: 1,
					Email: models.MatchOptions{
						MatchType: models.PartialMatchType,
						Value:     "@example",
					},
				},
			},
			"Email (Start with)": {
				Validate: func(s *UsersModelTestSuite, result models.PaginationData[objects.User]) {

					var expected = models.PaginationData[objects.User]{
						Page:       1,
						TotalPages: 2,
						Count:      3,
						Data:       users[:2],
					}

					s.Equal(expected, result)
				},
				Option: SearchOptions{
					CurrentPage: 1,
					Email: models.MatchOptions{
						MatchType: models.StartWithMatchType,
						Value:     "mail_search_",
					},
				},
			},
			"Email (End with)": {
				Validate: func(s *UsersModelTestSuite, result models.PaginationData[objects.User]) {

					var expected = models.PaginationData[objects.User]{
						Page:       1,
						TotalPages: 2,
						Count:      3,
						Data:       users[:2],
					}

					s.Equal(expected, result)
				},
				Option: SearchOptions{
					CurrentPage: 1,
					Email: models.MatchOptions{
						MatchType: models.EndWithMatchType,
						Value:     ".mock",
					},
				},
			},
		}

		for optionName, testCase := range testCases {

			s.Run(fmt.Sprintf("Search with option %s", optionName), func() {

				paginationData, err := s.model.Search(testCase.Option)
				s.Require().NoError(err, "Searching user failed")

				testCase.Validate(s, paginationData)
			})
		}
	})

	s.Run("Should throw error when set current page as non-positive value", func() {

		result, err := s.model.Search(SearchOptions{CurrentPage: 0})
		s.Require().ErrorIs(errors.CurrentPageInvalidError.New(), err, "Should have returned error")
		s.Require().Empty(result)
	})

	s.Run("Should throw error when set invalid or unsupported match type", func() {

		result, err := s.model.Search(
			SearchOptions{
				CurrentPage: 1,
				Username: models.MatchOptions{
					MatchType: 255,
				},
			},
		)
		s.Require().Error(err, "Should have returned error")
		s.Require().Empty(result)
	})

}

func (s *UsersModelTestSuite) TestUpdate() {

	s.Run("Should update exist user properly", func() {

		userToUpdate := mockUser()
		userToUpdate.UserID = s.insertedUser.UserID

		s.Require().NoError(s.model.Update(userToUpdate))
		s.insertedUser = userToUpdate

		actual, err := s.model.GetByID(userToUpdate.UserID)
		s.Require().NoError(err, "Getting updated user failed")
		s.Require().EqualValues(userToUpdate, actual)
	})

	s.Run("Should update partial data in user properly", func() {

		mockUser := mockUser()

		var userToUpdate = objects.User{
			UserID:   s.insertedUser.UserID,
			Password: mockUser.Password,
			Email:    mockUser.Email,
		}

		s.Require().NoError(s.model.Update(userToUpdate))

		expected := s.insertedUser
		expected.Password = userToUpdate.Password
		expected.Email = userToUpdate.Email

		s.insertedUser = expected

		actual, err := s.model.GetByID(expected.UserID)
		s.Require().NoError(err, "Getting updated user failed")
		s.Require().EqualValues(expected, actual)
	})

	s.Run("Should throw error when update non-exist user", func() {

		userToUpdate := mockUser()
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

func mockUser() objects.User {

	now := time.Now().UnixNano()

	return objects.User{
		UserID:      gofakeit.UUID(),
		Username:    fmt.Sprintf("username_%d", now),
		Password:    fmt.Sprintf("password_%d", now),
		AccountName: fmt.Sprintf("acct_%d", now),
		Email:       fmt.Sprintf("mail_%d@example.com", now),
	}
}
