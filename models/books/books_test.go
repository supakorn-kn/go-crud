package books

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/suite"
	"github.com/supakorn-kn/go-crud/errors"
	"github.com/supakorn-kn/go-crud/models"
	"github.com/supakorn-kn/go-crud/mongodb"
	"github.com/supakorn-kn/go-crud/objects"
	"go.mongodb.org/mongo-driver/bson"
)

type BooksModelTestSuite struct {
	suite.Suite
	conn         *mongodb.MongoDBConn
	model        *BooksModel
	insertedBook objects.Book
}

func (s *BooksModelTestSuite) SetupSuite() {

	//TODO: Will set parameter from env
	conn := mongodb.New("mongodb://localhost:27017", "go-crud_test")
	s.Require().NoError(conn.Connect(), "Create Mongodb connection failed")

	booksModel, err := NewBooksModel(&conn)
	if err != nil {
		defer conn.Disconnect()
		s.FailNow("Setup Book model failed", err)
	}

	s.model = booksModel
	s.conn = &conn
}

func (s *BooksModelTestSuite) BeforeTest(suiteName, testName string) {

	if testName == "TestInsert" || testName == "TestSearch" {
		return
	}

	s.insertedBook = fakeBook()
	s.Require().NoError(s.model.Insert(s.insertedBook), "Setup test failed from inserting book")

	_, err := s.model.GetByID(s.insertedBook.BookID)
	s.Require().NoError(err, "Setup test failed from inserting book")
}

func (s *BooksModelTestSuite) TearDownSuite() {
	s.conn.Disconnect()
}

func (s *BooksModelTestSuite) TestInsert() {

	s.Run("Should insert valid book properly", func() {

		book := fakeBook()

		err := s.model.Insert(book)
		s.Require().NoError(err, "Inserting Book failed")

		result := s.model.Coll.FindOne(context.Background(), bson.D{{Key: "book_id", Value: book.BookID}})

		var actual objects.Book
		s.Require().NoError(result.Decode(&actual), "Unmarshalling inserted Book failed")
		s.Require().EqualValues(book, actual, "Read data is not the same as inserted")
	})

	s.Run("Should throw error when insert book with existed book ID", func() {

		book := fakeBook()

		err := s.model.Insert(book)
		s.Require().NoError(err, "Inserting Book failed")

		newBook := fakeBook()
		newBook.BookID = book.BookID
		err = s.model.Insert(newBook)
		s.Require().Error(err, "Should have thrown errorr")
	})

	s.Run("Should throw error when insert invalid book data", func() {

		book := fakeBook()

		s.Run("Use empty book ID", func() {

			invalidBook := book
			invalidBook.BookID = ""

			err := s.model.Insert(invalidBook)
			s.Require().Error(err, "Should throw error")
		})

		s.Run("Use empty title", func() {

			invalidBook := book
			invalidBook.Title = ""

			err := s.model.Insert(invalidBook)
			s.Require().Error(err, "Should throw error")
		})

		s.Run("Use empty author", func() {

			invalidBook := book
			invalidBook.Author = ""

			err := s.model.Insert(invalidBook)
			s.Require().Error(err, "Should throw error")
		})

		s.Run("Use nil categories", func() {

			invalidBook := fakeBook()
			invalidBook.Categories = nil

			err := s.model.Insert(invalidBook)
			s.Require().Error(err, "Should throw error")
		})
	})
}

func (s *BooksModelTestSuite) TestGetByID() {

	s.Run("Should get the book by book_id properly", func() {

		actual, err := s.model.GetByID(s.insertedBook.BookID)
		s.Require().NoError(err, "Delete exist book failed")
		s.Require().EqualValues(s.insertedBook, actual)
	})

	s.Run("Should throw the error when give non-exist book_id", func() {

		itemID := "non-exist_id"

		actual, err := s.model.GetByID(itemID)
		s.Require().Equal(errors.ObjectIDNotFoundError.New(itemID), err, "Should throw error")
		s.Require().Empty(actual)
	})
}

func (s *BooksModelTestSuite) TestSearch() {

	bookA := objects.Book{
		BookID:      "book_search_0",
		Title:       "Search_A_Title",
		Author:      "Search_A_Author",
		Description: "First book",
		Categories:  []string{"Category A"},
	}
	bookB := objects.Book{
		BookID:      "book_search_1",
		Title:       "Search_B_Title",
		Author:      "Search_A_Author",
		Description: "book_search_0 author",
		Categories:  []string{"Category B"},
	}
	bookC := objects.Book{
		BookID:      "book_search_3",
		Title:       "Search_A_Title",
		Author:      "Search_B_Author",
		Description: "book_1 title and book_0,book_1 categories",
		Categories:  []string{"Category A", "Category B"},
	}

	sortedBooks := []objects.Book{bookA, bookC, bookB}

	shuffledBooks := slices.Clone[[]objects.Book, objects.Book](sortedBooks)
	gofakeit.ShuffleAnySlice(shuffledBooks)

	for _, book := range shuffledBooks {
		s.Require().NoError(s.model.Insert(book), "Insert books before testing failed")
	}

	var initialLimit = s.model.SearchLenLimit
	s.model.SearchLenLimit = 2

	s.T().Cleanup(func() {
		s.model.SearchLenLimit = initialLimit
	})

	totalDocumentsCount, err := s.model.Coll.CountDocuments(context.Background(), bson.D{})
	s.Require().NoError(err)
	totalAllDocumentsPage := totalDocumentsCount / int64(s.model.SearchLenLimit)
	if totalDocumentsCount%int64(s.model.SearchLenLimit) != 0 {
		totalAllDocumentsPage++
	}

	s.Run("Should get book properly by given options", func() {

		var testCases = map[string]struct {
			Validate func(*BooksModelTestSuite, models.PaginationData[objects.Book])
			Option   SearchOption
		}{
			"None (Page 1)": {
				Validate: func(s *BooksModelTestSuite, result models.PaginationData[objects.Book]) {

					s.Equal(1, result.Page)
					s.EqualValues(totalAllDocumentsPage, result.TotalPages)
					s.Len(result.Data, 2)
				},
				Option: SearchOption{
					CurrentPage: 1,
				},
			},
			"None (Page 2)": {
				Validate: func(s *BooksModelTestSuite, result models.PaginationData[objects.Book]) {
					s.Equal(2, result.Page)
					s.EqualValues(totalAllDocumentsPage, result.TotalPages)

					dataLen := len(result.Data)

					s.GreaterOrEqual(dataLen, 1)
					s.Less(dataLen, 3)
				},
				Option: SearchOption{
					CurrentPage: 2,
				},
			},
			"Title (Equal)": {
				Validate: func(s *BooksModelTestSuite, result models.PaginationData[objects.Book]) {
					s.Equal(1, result.Page)
					s.EqualValues(1, result.TotalPages)
					s.EqualValues([]objects.Book{bookA, bookC}, result.Data)
				},
				Option: SearchOption{
					CurrentPage: 1,
					Title: models.MatchOption{
						MatchType: models.EqualMatchType,
						Value:     bookA.Title,
					},
				},
			},
			"Title (Partial)": {
				Validate: func(s *BooksModelTestSuite, result models.PaginationData[objects.Book]) {
					s.Equal(1, result.Page)
					s.EqualValues(1, result.TotalPages)
					s.EqualValues([]objects.Book{bookB}, result.Data)
				},
				Option: SearchOption{
					CurrentPage: 1,
					Title: models.MatchOption{
						MatchType: models.PartialMatchType,
						Value:     "search_b",
					},
				},
			},
			"Title (Start with)": {
				Validate: func(s *BooksModelTestSuite, result models.PaginationData[objects.Book]) {
					s.Equal(1, result.Page)
					s.EqualValues(2, result.TotalPages)
					s.EqualValues(sortedBooks[0:2], result.Data)
				},
				Option: SearchOption{
					CurrentPage: 1,
					Title: models.MatchOption{
						MatchType: models.StartWithMatchType,
						Value:     "search_",
					},
				},
			},
			"Title (End with)": {
				Validate: func(s *BooksModelTestSuite, result models.PaginationData[objects.Book]) {
					s.Equal(1, result.Page)
					s.EqualValues(1, result.TotalPages)
					s.EqualValues([]objects.Book{bookA, bookC}, result.Data)
				},
				Option: SearchOption{
					CurrentPage: 1,
					Title: models.MatchOption{
						MatchType: models.EndWithMatchType,
						Value:     "a_title",
					},
				},
			},
			"Author (Equal)": {
				Validate: func(s *BooksModelTestSuite, result models.PaginationData[objects.Book]) {
					s.Equal(1, result.Page)
					s.EqualValues(1, result.TotalPages)
					s.EqualValues([]objects.Book{bookA, bookB}, result.Data)
				},
				Option: SearchOption{
					CurrentPage: 1,
					Author: models.MatchOption{
						MatchType: models.EqualMatchType,
						Value:     bookA.Author,
					},
				},
			},
			"Author (Partial)": {
				Validate: func(s *BooksModelTestSuite, result models.PaginationData[objects.Book]) {
					s.Equal(1, result.Page)
					s.EqualValues(1, result.TotalPages)
					s.EqualValues([]objects.Book{bookC}, result.Data)
				},
				Option: SearchOption{
					CurrentPage: 1,
					Author: models.MatchOption{
						MatchType: models.PartialMatchType,
						Value:     "search_b",
					},
				},
			},
			"Author (Start with)": {
				Validate: func(s *BooksModelTestSuite, result models.PaginationData[objects.Book]) {
					s.Equal(1, result.Page)
					s.EqualValues(2, result.TotalPages)
					s.EqualValues(sortedBooks[0:2], result.Data)
				},
				Option: SearchOption{
					CurrentPage: 1,
					Author: models.MatchOption{
						MatchType: models.StartWithMatchType,
						Value:     "search_",
					},
				},
			},
			"Author (End with)": {
				Validate: func(s *BooksModelTestSuite, result models.PaginationData[objects.Book]) {
					s.Equal(1, result.Page)
					s.EqualValues(1, result.TotalPages)
					s.EqualValues([]objects.Book{bookA, bookB}, result.Data)
				},
				Option: SearchOption{
					CurrentPage: 1,
					Author: models.MatchOption{
						MatchType: models.EndWithMatchType,
						Value:     "a_author",
					},
				},
			},
		}

		for optionName, testCase := range testCases {

			s.Run(fmt.Sprintf("Search with option %s", optionName), func() {

				paginationData, err := s.model.Search(testCase.Option)
				s.Require().NoError(err, "Searching book failed")

				testCase.Validate(s, paginationData)
			})
		}
	})

	s.Run("Should throw error when set current page as non-positive value", func() {

		result, err := s.model.Search(SearchOption{CurrentPage: 0})
		s.Require().Equal(errors.CurrentPageInvalidError.New(), err, "Should have returned error")
		s.Require().Empty(result)
	})

	s.Run("Should throw error when set invalid or unsupported match type", func() {

		result, err := s.model.Search(
			SearchOption{
				CurrentPage: 1,
				Title: models.MatchOption{
					MatchType: 255,
				},
			},
		)
		s.Require().Error(err, "Should have returned error")
		s.Require().Empty(result)
	})
}

func (s *BooksModelTestSuite) TestUpdate() {

	s.Run("Should update exist book properly", func() {

		bookToUpdate := fakeBook()
		bookToUpdate.BookID = s.insertedBook.BookID

		s.Require().NoError(s.model.Update(bookToUpdate))

		s.T().Cleanup(func() {
			s.Require().NoError(s.model.Update(s.insertedBook))
		})

		actual, err := s.model.GetByID(bookToUpdate.BookID)
		s.Require().NoError(err, "Getting updated book failed")
		s.Require().EqualValues(bookToUpdate, actual)
	})

	s.Run("Should update partial data in book properly", func() {

		mockBook := fakeBook()

		var bookToUpdate = objects.Book{
			BookID:      s.insertedBook.BookID,
			Author:      mockBook.Author,
			Description: mockBook.Description,
		}

		s.Require().NoError(s.model.Update(bookToUpdate))

		s.T().Cleanup(func() {
			s.Require().NoError(s.model.Update(s.insertedBook))
		})

		expected := s.insertedBook
		expected.Author = bookToUpdate.Author
		expected.Description = bookToUpdate.Description

		actual, err := s.model.GetByID(expected.BookID)
		s.Require().NoError(err, "Getting updated book failed")
		s.Require().EqualValues(expected, actual)
	})

	s.Run("Should throw error when update non-exist book", func() {

		bookToUpdate := fakeBook()
		s.Require().Error(s.model.Update(bookToUpdate))

		actual, err := s.model.GetByID(s.insertedBook.BookID)
		s.Require().NoError(err, "Getting updated book failed")
		s.Require().EqualValues(s.insertedBook, actual)
	})
}

func (s *BooksModelTestSuite) TestDelete() {

	s.Run("Should delete exist book properly", func() {

		s.Require().NoError(s.model.Delete(s.insertedBook.BookID), "Delete exist book failed")

		actual, err := s.model.GetByID(s.insertedBook.BookID)
		s.Require().Error(err, "Should throw error after getting deleted book")
		s.Require().Empty(actual, "The book should have been empty")
	})

	s.Run("Should throw error when delete non-exist book", func() {

		s.Require().Error(s.model.Delete(s.insertedBook.BookID), "Delete exist book failed")
	})
}

func TestBooksModel(t *testing.T) {
	suite.Run(t, new(BooksModelTestSuite))
}

func fakeBook() objects.Book {

	fakeInfo := gofakeit.Book()
	uuid := gofakeit.UUID()

	return objects.Book{
		BookID:      fmt.Sprintf("book_%s", uuid),
		Title:       fakeInfo.Title,
		Author:      fakeInfo.Author,
		Description: gofakeit.SentenceSimple(),
		Categories:  []string{fakeInfo.Genre},
	}
}
