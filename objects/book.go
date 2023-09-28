package objects

import "reflect"

type Book struct {
	BookID      string   `json:"book_id" bson:"book_id,omitempty"`
	Title       string   `json:"title" bson:"title,omitempty"`
	Author      string   `json:"author" bson:"author,omitempty"`
	Description string   `json:"description" bson:"description,omitempty"`
	Categories  []string `json:"categories" bson:"categories"`
}

func (b Book) GetID() string {
	return b.BookID
}

func (b Book) IsNil() bool {
	return reflect.ValueOf(b).IsZero()
}
