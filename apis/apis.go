package apis

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/supakorn-kn/go-crud/errors"
	"github.com/supakorn-kn/go-crud/models"
)

type CRUDResponse struct {
	Result any              `json:"result,omitempty"`
	Error  errors.BaseError `json:"error,omitempty"`
}

func (resp CRUDResponse) MarshalJSON() ([]byte, error) {

	data := map[string]any{"error": resp.Error}

	if resp.Error.IsNil() {
		data = map[string]any{"result": resp.Result}
	}

	return json.Marshal(data)
}

type CrudAPI[Item models.Item] interface {
	Insert(ctx *gin.Context) error
	ReadOne(itemID string, ctx *gin.Context) (*Item, error)
	Read(ctx *gin.Context) (*models.PaginationData[Item], error)
	Update(ctx *gin.Context) error
	Delete(ctx *gin.Context) error
}

var OKResponse = CRUDResponse{Result: map[string]any{"status": "OK"}}
