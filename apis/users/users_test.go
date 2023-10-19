package users

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
	"github.com/supakorn-kn/go-crud/apis"
	"github.com/supakorn-kn/go-crud/errors"
	"github.com/supakorn-kn/go-crud/models"
	"github.com/supakorn-kn/go-crud/models/users"
	"github.com/supakorn-kn/go-crud/mongodb"
	"github.com/supakorn-kn/go-crud/objects"
)

type UsersAPISuite struct {
	suite.Suite
	conn        mongodb.MongoDBConn
	api         *UsersCrudAPI
	g           *gin.Engine
	createdUser objects.User
}

func (s *UsersAPISuite) SetupSuite() {

	conn := mongodb.New("mongodb://localhost:27017", "go-crud_test")
	s.Require().NoError(conn.Connect(), "Create MongoDB connection failed")

	s.conn = conn
	api, err := NewUsersAPI(&conn)
	if err != nil {
		s.conn.Disconnect()
		s.FailNow("Create user API failed", err)
	}

	g := gin.Default()
	apis.RegisterCrudAPI[objects.User](api, g.Group("api/users"))

	s.g = g
	s.api = api
}

func (s *UsersAPISuite) BeforeTest(suiteName, testName string) {

	if testName == "TestCreate" || testName == "TestRead" {
		return
	}

	user := mockUser()
	s.Require().NoError(s.api.model.Insert(user), "Inserting user before testing failed")

	s.createdUser = user
}

func (s *UsersAPISuite) AfterTest(suiteName, testName string) {

	if testName == "TestDelete" {
		return
	}

	s.Require().NoError(s.api.model.Delete(s.createdUser.UserID), "Clearing after tested failed from inserting user")
}

func (s *UsersAPISuite) TearDownSuite() {
	s.conn.Disconnect()
}

func (s *UsersAPISuite) TestCreate() {

	user := mockUser()
	s.createdUser = user

	createUserFunc := func(user objects.User) *httptest.ResponseRecorder {

		b, err := json.Marshal(user)
		s.Require().NoError(err)

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/users", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)
		return recorder
	}

	s.Run("Should create user properly", func() {

		recorder := createUserFunc(user)
		s.Require().Equal(http.StatusCreated, recorder.Code)

		var resp apis.CRUDResponse
		s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &resp))
		s.Empty(resp.Error)
		s.Equal(apis.OKResponse, resp)
	})

	s.Run("Should throw error when create user using incomplete filled user data", func() {

		newUser := mockUser()
		newUser.Username = ""

		recorder := createUserFunc(newUser)
		s.Require().Equal(http.StatusBadRequest, recorder.Code)

		var resp apis.CRUDResponse
		s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &resp))
		s.Empty(resp.Result)
		s.True(errors.DataValidationFailedError.IsEqual(resp.Error))
	})

	s.Run("Should throw error when create user using existed user_id", func() {

		recorder := createUserFunc(user)
		s.Require().Equal(http.StatusBadRequest, recorder.Code)

		var resp apis.CRUDResponse
		s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &resp))
		s.Empty(resp.Result)
		s.True(errors.DataAlreadyInUsedError.IsEqual(resp.Error))
	})
}

func (s *UsersAPISuite) TestRead() {

	user := objects.User{
		UserID:      "user_for_test_read",
		Username:    "test_read_username",
		Password:    "test_read_passwd",
		AccountName: "test_read_account_name",
		Email:       "test_read@example.mock",
	}
	s.Require().NoError(s.api.model.Insert(user), "Inserting user before testing failed")

	s.createdUser = user

	s.Run("Should read user properly", func() {

		searchOptions := users.SearchOptions{
			CurrentPage: 1,
			Username: models.MatchOptions{
				MatchType: 0,
				Value:     user.Username,
			},
			AccountName: models.MatchOptions{
				MatchType: 0,
				Value:     user.AccountName,
			},
		}

		b, err := json.Marshal(searchOptions)
		s.Require().NoError(err)

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/users", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)
		s.Require().Equal(http.StatusOK, recorder.Code)

		var expected, _ = json.Marshal(apis.CRUDResponse{
			Result: models.PaginationData[objects.User]{
				Page:       1,
				TotalPages: 1,
				Count:      1,
				Data:       []objects.User{user},
			},
		})

		s.JSONEq(string(expected), recorder.Body.String())
	})

	s.Run("Should throw error when user does not give search options", func() {

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/users", nil)
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)
		s.Require().Equal(http.StatusBadRequest, recorder.Code)

		var resp apis.CRUDResponse
		s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &resp))
		s.NotEmpty(resp.Error)
		s.Empty(resp.Result)
	})

	s.Run("Should throw error when user give impossible match type value (out of uint8 range)", func() {

		searchOptions := map[string]any{
			"current_page": 1,
			"username": map[string]any{
				"match_type": -1,
				"value":      user.Username,
			},
		}
		b, err := json.Marshal(searchOptions)
		s.Require().NoError(err)

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/users", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)
		s.Require().Equal(http.StatusBadRequest, recorder.Code)

		var resp apis.CRUDResponse
		s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &resp))
		s.NotEmpty(resp.Error)
		s.Empty(resp.Result)
	})

	s.Run("Should throw error when user does not fill current page (current page = 0) in search options", func() {

		searchOptions := users.SearchOptions{
			CurrentPage: 0,
			Username: models.MatchOptions{
				MatchType: 0,
				Value:     user.Username,
			},
		}
		b, err := json.Marshal(searchOptions)
		s.Require().NoError(err)

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/users", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)
		s.Require().Equal(http.StatusBadRequest, recorder.Code)

		var resp apis.CRUDResponse
		s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &resp))
		s.Empty(resp.Result)
		s.True(errors.CurrentPageInvalidError.IsEqual(resp.Error))
	})
}

func (s *UsersAPISuite) TestReadOne() {

	s.Run("Should get user by user_id properly", func() {

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/users/%s", s.createdUser.UserID), nil)
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)
		s.Require().Equal(http.StatusOK, recorder.Code)

		var expected, _ = json.Marshal(apis.CRUDResponse{Result: s.createdUser})

		var resp apis.CRUDResponse
		s.Empty(resp.Error)
		s.JSONEq(string(expected), recorder.Body.String())
	})

	s.Run("Should throw error when user give invalid user ID", func() {

		itemID := "invalid_id"

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/users/%s", itemID), nil)
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)
		s.Require().Equal(http.StatusNotFound, recorder.Code)

		var resp apis.CRUDResponse
		s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &resp))
		s.Empty(resp.Result)
		s.True(errors.ObjectIDNotFoundError.IsEqual(resp.Error))
	})
}

func (s *UsersAPISuite) TestUpdate() {

	s.Run("Should update user properly", func() {

		user := mockUser()
		user.UserID = s.createdUser.UserID

		b, err := json.Marshal(user)
		s.Require().NoError(err)

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPut, "/api/users", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)
		s.Require().Equal(http.StatusNoContent, recorder.Code)
		s.Empty(recorder.Body.Bytes())
	})

	s.Run("Should throw error when user give invalid user ID", func() {

		user := mockUser()

		b, err := json.Marshal(user)
		s.Require().NoError(err)

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPut, "/api/users", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)
		s.Require().Equal(http.StatusNotFound, recorder.Code)

		var resp apis.CRUDResponse
		s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &resp))
		s.Empty(resp.Result)
		s.True(errors.ObjectIDNotFoundError.IsEqual(resp.Error))
	})
}

func (s *UsersAPISuite) TestDelete() {

	s.Run("Should delete user properly", func() {

		b, err := json.Marshal(s.createdUser)
		s.Require().NoError(err)

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodDelete, "/api/users", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)
		s.Require().Equal(http.StatusNoContent, recorder.Code)
		s.Empty(recorder.Body.Bytes())
	})

	s.Run("Should throw error when user give invalid user ID", func() {

		user := mockUser()

		b, err := json.Marshal(user)
		s.Require().NoError(err)

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPut, "/api/users", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)
		s.Require().Equal(http.StatusNotFound, recorder.Code)

		var resp apis.CRUDResponse
		s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &resp))
		s.Empty(resp.Result)
		s.True(errors.ObjectIDNotFoundError.IsEqual(resp.Error))
	})
}

func TestUsersAPI(t *testing.T) {
	suite.Run(t, new(UsersAPISuite))
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
