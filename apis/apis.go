package apis

import (
	"github.com/gin-gonic/gin"
	"github.com/supakorn-kn/go-crud/models"
)

type CrudAPI[Item any] interface {
	Insert(ctx *gin.Context) error
	ReadOne(itemID string, ctx *gin.Context) (*Item, error)
	Read(ctx *gin.Context) (*models.PaginationData[Item], error)
	Update(ctx *gin.Context) error
	Delete(ctx *gin.Context) error
}

var OKResponse = map[string]string{"status": "OK"}
