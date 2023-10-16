package books

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
	"github.com/supakorn-kn/go-crud/models/books"
	"github.com/supakorn-kn/go-crud/mongodb"
	"github.com/supakorn-kn/go-crud/objects"
)

type BooksAPISuite struct {
	suite.Suite
	conn        mongodb.MongoDBConn
	api         *BooksCrudAPI
	g           *gin.Engine
	createdBook objects.Book
}

func (s *BooksAPISuite) SetupSuite() {

	conn := mongodb.New("mongodb://localhost:27017", "go-crud_test")
	err := conn.Connect()
	if err != nil {
		s.Require().Fail("Create MongoDB connection failed", err)
	}

	s.conn = conn
	api, err := NewBooksAPI(&conn)
	if err != nil {
		s.conn.Disconnect()
		s.Require().Fail("Create books API failed", err)
	}

	g := gin.Default()
	apis.RegisterCrudAPI[objects.Book](api, g.Group("api/books"))

	s.g = g
	s.api = api
}

func (s *BooksAPISuite) BeforeTest(suiteName, testName string) {

	if testName == "TestCreate" {
		return
	}

	book := fakeBook()
	s.Require().NoError(s.api.model.Insert(book), "Inserting book before testing failed")

	s.createdBook = book
}

func (s *BooksAPISuite) AfterTest(suiteName, testName string) {

	if testName == "TestCreate" || testName == "TestDelete" {
		return
	}

	s.Require().NoError(s.api.model.Delete(s.createdBook.BookID), "Clearing after tested failed from inserting book")
}

func (s *BooksAPISuite) TearDownSuite() {
	s.conn.Disconnect()
}

func (s *BooksAPISuite) TestCreate() {

	book := fakeBook()

	createBookFunc := func(book objects.Book) *httptest.ResponseRecorder {

		b, err := json.Marshal(book)
		s.Require().NoError(err)

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/books", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)
		return recorder
	}

	s.Run("Should create book properly", func() {

		recorder := createBookFunc(book)
		s.Require().Equal(http.StatusCreated, recorder.Code)

		var resp apis.CRUDResponse
		s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &resp))
		s.Empty(resp.Error)
		s.Equal(apis.OKResponse, resp)
	})

	s.Run("Should throw error when create book using incomplete filled book data", func() {

		newBook := fakeBook()
		newBook.Title = ""

		recorder := createBookFunc(newBook)
		s.Require().Equal(http.StatusBadRequest, recorder.Code)

		var resp apis.CRUDResponse
		s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &resp))
		s.Empty(resp.Result)
		s.True(errors.IsError(resp.Error, errors.DataValidationFailedError.New()))
	})

	s.Run("Should throw error when create book using existed book_id", func() {

		recorder := createBookFunc(book)
		s.Require().Equal(http.StatusBadRequest, recorder.Code)

		var resp apis.CRUDResponse
		s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &resp))
		s.Empty(resp.Result)
		s.True(errors.IsError(resp.Error, errors.DataAlreadyInUsedError.New()))
	})
}

func (s *BooksAPISuite) TestRead() {

	s.Run("Should create book properly", func() {

		searchOption := books.SearchOption{
			CurrentPage: 1,
			Title: models.MatchOption{
				MatchType: 0,
				Value:     s.createdBook.Title,
			},
			Author: models.MatchOption{
				MatchType: 0,
				Value:     s.createdBook.Author,
			},
		}

		b, err := json.Marshal(searchOption)
		s.Require().NoError(err)

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/books", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)
		s.Require().Equal(http.StatusOK, recorder.Code)

		var expected, _ = json.Marshal(apis.CRUDResponse{
			Result: models.PaginationData[objects.Book]{
				Page:       1,
				TotalPages: 1,
				Data:       []objects.Book{s.createdBook},
			},
		})

		var resp apis.CRUDResponse
		s.Empty(resp.Error)
		s.JSONEq(string(expected), recorder.Body.String())
	})

	s.Run("Should throw error when user does not give search option", func() {

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/books", nil)
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)
		s.Require().Equal(http.StatusBadRequest, recorder.Code)

		var resp apis.CRUDResponse
		s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &resp))
		s.NotEmpty(resp.Error)
		s.Empty(resp.Result)
	})

	s.Run("Should throw error when user give impossible match type value (out of uint8 range)", func() {

		searchOption := map[string]any{
			"current_page": 1,
			"title": map[string]any{
				"match_type": -1,
				"value":      s.createdBook.Title,
			},
		}
		b, err := json.Marshal(searchOption)
		s.Require().NoError(err)

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/books", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)
		s.Require().Equal(http.StatusBadRequest, recorder.Code)

		var resp apis.CRUDResponse
		s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &resp))
		s.NotEmpty(resp.Error)
		s.Empty(resp.Result)
	})

	s.Run("Should throw error when user does not fill current page (current page = 0) in search option", func() {

		searchOption := books.SearchOption{}
		b, err := json.Marshal(searchOption)
		s.Require().NoError(err)

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/books", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)
		s.Require().Equal(http.StatusBadRequest, recorder.Code)

		var resp apis.CRUDResponse
		s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &resp))
		s.Empty(resp.Result)
		s.True(errors.IsError(resp.Error, errors.CurrentPageInvalidError.New()))
	})
}

func (s *BooksAPISuite) TestReadOne() {

	s.Run("Should get book by book_id properly", func() {

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/books/%s", s.createdBook.BookID), nil)
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)
		s.Require().Equal(http.StatusOK, recorder.Code)

		var expected, _ = json.Marshal(apis.CRUDResponse{Result: s.createdBook})

		var resp apis.CRUDResponse
		s.Empty(resp.Error)
		s.JSONEq(string(expected), recorder.Body.String())
	})

	s.Run("Should throw error when user give invalid book ID", func() {

		itemID := "invalid_id"

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/books/%s", itemID), nil)
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)
		s.Require().Equal(http.StatusNotFound, recorder.Code)

		var resp apis.CRUDResponse
		s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &resp))
		s.Empty(resp.Result)
		s.True(errors.IsError(resp.Error, errors.ObjectIDNotFoundError.New(itemID)))
	})
}

func (s *BooksAPISuite) TestUpdate() {

	s.Run("Should update book properly", func() {

		book := fakeBook()
		book.BookID = s.createdBook.BookID

		b, err := json.Marshal(book)
		s.Require().NoError(err)

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPut, "/api/books", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)
		s.Require().Equal(http.StatusNoContent, recorder.Code)
		s.Empty(recorder.Body.Bytes())
	})

	s.Run("Should throw error when user give invalid book ID", func() {

		book := fakeBook()

		b, err := json.Marshal(book)
		s.Require().NoError(err)

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPut, "/api/books", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)
		s.Require().Equal(http.StatusNotFound, recorder.Code)

		var resp apis.CRUDResponse
		s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &resp))
		s.Empty(resp.Result)
		s.True(errors.IsError(resp.Error, errors.ObjectIDNotFoundError.New(book.BookID)))
	})
}

func (s *BooksAPISuite) TestDelete() {

	s.Run("Should delete book properly", func() {

		b, err := json.Marshal(s.createdBook)
		s.Require().NoError(err)

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodDelete, "/api/books", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)
		s.Require().Equal(http.StatusNoContent, recorder.Code)
		s.Empty(recorder.Body.Bytes())
	})

	s.Run("Should throw error when user give invalid book ID", func() {

		book := fakeBook()

		b, err := json.Marshal(book)
		s.Require().NoError(err)

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPut, "/api/books", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)
		s.Require().Equal(http.StatusNotFound, recorder.Code)

		var resp apis.CRUDResponse
		s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &resp))
		s.Empty(resp.Result)
		s.True(errors.IsError(resp.Error, errors.ObjectIDNotFoundError.New(book.BookID)))
	})
}

func TestBooksAPI(t *testing.T) {
	suite.Run(t, new(BooksAPISuite))
}

func fakeBook() objects.Book {

	fakeInfo := gofakeit.Book()

	now := time.Now()

	return objects.Book{
		BookID:      "book_" + fmt.Sprintf("%d", now.UnixNano()),
		Title:       fakeInfo.Title,
		Author:      fakeInfo.Author,
		Description: gofakeit.SentenceSimple(),
		Categories:  []string{fakeInfo.Genre},
	}
}
