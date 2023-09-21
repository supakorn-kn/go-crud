package books

import (
	"bytes"
	"context"
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
	"github.com/supakorn-kn/go-crud/models"
	"github.com/supakorn-kn/go-crud/models/books"
	"github.com/supakorn-kn/go-crud/mongodb"
	"github.com/supakorn-kn/go-crud/objects"
	"go.mongodb.org/mongo-driver/bson"
)

type BooksAPISuite struct {
	suite.Suite
	conn        mongodb.MongoDBConn
	g           *gin.Engine
	createdBook objects.Book
}

func (s *BooksAPISuite) SetupSuite() {

	conn := mongodb.New("mongodb://localhost:27017", "go-crud_test")
	err := conn.Connect()
	if err != nil {
		s.Require().Fail("Create Mongodb connection failed", err)
	}

	s.conn = conn
	api, err := NewBooksAPI(&conn)
	if err != nil {
		s.conn.GetDatabase().Drop(context.Background())
		s.conn.Disconnect()

		s.Require().Fail("Create books API failed", err)
	}

	gin.SetMode(gin.TestMode)

	g := gin.Default()
	apis.RegisterCrudAPI[objects.Book]("api/books", api, g)

	s.g = g
}

func (s *BooksAPISuite) BeforeTest(suiteName, testName string) {

	if testName == "TestCreate" {
		return
	}

	book := fakeBook()
	recorder := createBook(s, book)
	s.Require().Equal(http.StatusCreated, recorder.Code, "Creating book before test failed")

	s.createdBook = book
}

func (s *BooksAPISuite) AfterTest(suiteName, testName string) {

	collectionName := books.BooksModel{}.GetCollectionName()

	_, err := s.conn.GetCollection(collectionName).DeleteMany(context.Background(), bson.D{})
	s.Require().NoError(err)
}

func (s *BooksAPISuite) TearDownSuite() {

	s.conn.GetDatabase().Drop(context.Background())
	s.conn.Disconnect()
}

func (s *BooksAPISuite) TestCreate() {

	book := fakeBook()

	createBookStatements := func(book objects.Book) *httptest.ResponseRecorder {
		b, err := json.Marshal(book)
		s.Require().NoError(err)

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/books", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)

		return recorder
	}

	s.Run("Should create book properly", func() {

		recorder := createBookStatements(book)
		s.Require().Equal(http.StatusCreated, recorder.Code)

		var result map[string]string
		s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &result))
		s.Require().EqualValues(apis.OKResponse, result)
	})

	s.Run("Should throw error when create book using existed book_id", func() {

		recorder := createBookStatements(book)
		s.Require().Equal(http.StatusBadRequest, recorder.Code)

		//TODO: Assert error content
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

		var expected = models.PaginationData[objects.Book]{
			Page:       1,
			TotalPages: 1,
			Data:       []objects.Book{s.createdBook},
		}
		var result models.PaginationData[objects.Book]
		s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &result))
		s.Require().EqualValues(expected, result)
	})

	s.Run("Should throw error when user does not give search option", func() {

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/books", nil)
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)
		s.Require().Equal(http.StatusBadRequest, recorder.Code)

		//TODO: Assert error content
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

		//TODO: Assert error content
	})
}

func (s *BooksAPISuite) TestReadOne() {

	s.Run("Should get book by book_id properly", func() {

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/books/%s", s.createdBook.BookID), nil)
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)

		var result objects.Book
		s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &result))
		s.Require().EqualValues(s.createdBook, result)
	})

	s.Run("Should throw error when user give invalid book ID", func() {

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/books/invalid_id", nil)
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)
		s.Require().Equal(http.StatusBadRequest, recorder.Code)

		//TODO: Assert error content
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
		s.Require().Equal(http.StatusOK, recorder.Code)

		var result map[string]string
		s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &result))
		s.Require().EqualValues(apis.OKResponse, result)
	})

	s.Run("Should throw error when user give invalid book ID", func() {

		book := fakeBook()

		b, err := json.Marshal(book)
		s.Require().NoError(err)

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPut, "/api/books", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)
		s.Require().Equal(http.StatusBadRequest, recorder.Code)

		//TODO: Assert error content
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
		s.Require().Equal(http.StatusOK, recorder.Code)

		var result map[string]string
		s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &result))
		s.Require().EqualValues(apis.OKResponse, result)

		//TODO: Assert getting book by id should throw error
	})

	s.Run("Should throw error when user give invalid book ID", func() {

		book := fakeBook()

		b, err := json.Marshal(book)
		s.Require().NoError(err)

		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPut, "/api/books", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")

		s.g.ServeHTTP(recorder, req)
		s.Require().Equal(http.StatusBadRequest, recorder.Code)

		//TODO: Assert error content
	})
}

func TestBooksAPI(t *testing.T) {
	suite.Run(t, new(BooksAPISuite))
}

func createBook(s *BooksAPISuite, book objects.Book) *httptest.ResponseRecorder {

	b, err := json.Marshal(book)
	s.Require().NoError(err)

	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/books", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")

	s.g.ServeHTTP(recorder, req)

	return recorder
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
