package books

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

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
	s.Require().NoError(conn.Connect(), "Create MongoDB connection failed")

	s.conn = conn
	api, err := NewBooksAPI(&conn)
	if err != nil {
		s.conn.Disconnect()
		s.FailNow("Create books API failed", err)
	}

	g := gin.Default()
	apis.RegisterCrudAPI[objects.Book](api, g.Group("api/books"))

	s.g = g
	s.api = api
}

func (s *BooksAPISuite) BeforeTest(suiteName, testName string) {

	if testName == "TestCreate" || testName == "TestRead" {
		return
	}

	book := mockBook()
	s.Require().NoError(s.api.model.Insert(book), "Inserting book before testing failed")

	s.createdBook = book
}

func (s *BooksAPISuite) AfterTest(suiteName, testName string) {

	if testName == "TestDelete" {
		return
	}

	s.Require().NoError(s.api.model.Delete(s.createdBook.BookID), "Clearing after tested failed from inserting book")
}

func (s *BooksAPISuite) TearDownSuite() {
	s.conn.Disconnect()
}

func (s *BooksAPISuite) TestCreate() {

	book := mockBook()
	s.createdBook = book

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

		newBook := mockBook()
		newBook.Title = ""

		recorder := createBookFunc(newBook)
		s.Require().Equal(http.StatusBadRequest, recorder.Code)

		var resp apis.CRUDResponse
		s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &resp))
		s.Empty(resp.Result)
		s.True(errors.DataValidationFailedError.IsEqual(resp.Error))
	})

	s.Run("Should throw error when create book using existed book_id", func() {

		recorder := createBookFunc(book)
		s.Require().Equal(http.StatusBadRequest, recorder.Code)

		var resp apis.CRUDResponse
		s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &resp))
		s.Empty(resp.Result)
		s.True(errors.DataAlreadyInUsedError.IsEqual(resp.Error))
	})
}

func (s *BooksAPISuite) TestRead() {

	book := objects.Book{
		BookID:      "book_for_test_read",
		Title:       "test_read_title",
		Author:      "test_read_author",
		Description: "test_read_description",
		Categories:  []string{"test_read_category"},
	}
	s.Require().NoError(s.api.model.Insert(book), "Inserting book before testing failed")

	s.createdBook = book

	s.Run("Should read book properly", func() {

		searchOptions := books.SearchOptions{
			CurrentPage: 1,
			Title: models.MatchOptions{
				MatchType: 0,
				Value:     book.Title,
			},
			Author: models.MatchOptions{
				MatchType: 0,
				Value:     book.Author,
			},
		}

		b, err := json.Marshal(searchOptions)
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
				Count:      1,
				Data:       []objects.Book{book},
			},
		})

		s.JSONEq(string(expected), recorder.Body.String())
	})

	s.Run("Should throw error when user does not give search options", func() {

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

		searchOptions := map[string]any{
			"current_page": 1,
			"title": map[string]any{
				"match_type": -1,
				"value":      book.Title,
			},
		}
		b, err := json.Marshal(searchOptions)
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

	s.Run("Should throw error when user does not fill current page (current page = 0) in search options", func() {

		searchOptions := books.SearchOptions{
			CurrentPage: 0,
			Title: models.MatchOptions{
				MatchType: 0,
				Value:     book.Title,
			},
		}
		b, err := json.Marshal(searchOptions)
		s.Require().NoError(err)

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/books", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)
		s.Require().Equal(http.StatusBadRequest, recorder.Code)

		var resp apis.CRUDResponse
		s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &resp))
		s.Empty(resp.Result)
		s.True(errors.CurrentPageInvalidError.IsEqual(resp.Error))
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
		s.True(errors.ObjectIDNotFoundError.IsEqual(resp.Error))
	})
}

func (s *BooksAPISuite) TestUpdate() {

	s.Run("Should update book properly", func() {

		book := mockBook()
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

		book := mockBook()

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
		s.True(errors.ObjectIDNotFoundError.IsEqual(resp.Error))
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

		book := mockBook()

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
		s.True(errors.ObjectIDNotFoundError.IsEqual(resp.Error))
	})
}

func TestBooksAPI(t *testing.T) {
	suite.Run(t, new(BooksAPISuite))
}

func mockBook() objects.Book {

	fakeInfo := gofakeit.Book()

	return objects.Book{
		BookID:      fmt.Sprintf("book_%s", gofakeit.UUID()),
		Title:       fakeInfo.Title,
		Author:      fakeInfo.Author,
		Description: gofakeit.SentenceSimple(),
		Categories:  []string{fakeInfo.Genre},
	}
}
