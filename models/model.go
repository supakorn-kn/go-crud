package models

type Model[Item any] interface {
	GetCollectionName() string
	Insert(item Item) error
	GetByID(itemID string) (Item, error)
	Search() (PaginationData[Item], error)
	Update(item Item) error
	Delete(bookID string) error
}

type PaginationData[Data any] struct {
	Page       int    `json:"page"`
	TotalPages int    `json:"total_pages"`
	Data       []Data `json:"data"`
}
