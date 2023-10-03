package apis

import (
	"github.com/gin-gonic/gin"
	"github.com/supakorn-kn/go-crud/errors"
	"github.com/supakorn-kn/go-crud/models"
)

type CRUDResponse struct {
	Result any              `json:"result,omitempty"`
	Error  errors.BaseError `json:"error,omitempty"`
}

type CrudAPI[Item models.Item] interface {
	Insert(ctx *gin.Context) error
	ReadOne(itemID string, ctx *gin.Context) (*Item, error)
	Read(ctx *gin.Context) (*models.PaginationData[Item], error)
	Update(ctx *gin.Context) error
	Delete(ctx *gin.Context) error
}

var OKResponse = CRUDResponse{Result: map[string]any{"status": "OK"}}
