package apis

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/supakorn-kn/go-crud/errors"
)

const (
	responseContextKey = "response"
)

func RegisterCrudAPI[Item any](api CrudAPI[Item], group *gin.RouterGroup) {

	group.Use(func(ctx *gin.Context) {

		ctx.Next()
	})

	group.POST("", func(ctx *gin.Context) {

		err := api.Insert(ctx)
		if err != nil {
			writeErrorJSON(ctx, err)
			return
		}

		ctx.JSON(http.StatusCreated, OKResponse)
	})

	group.GET(":id", func(ctx *gin.Context) {

		itemID := ctx.Param("id")

		item, err := api.ReadOne(itemID, ctx)
		if err != nil {
			writeErrorJSON(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, CRUDResponse{Result: item})
	})

	group.GET("", func(ctx *gin.Context) {

		paginateResult, err := api.Read(ctx)

		if err != nil {
			writeErrorJSON(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, CRUDResponse{Result: paginateResult})
	})

	group.PUT("", func(ctx *gin.Context) {

		err := api.Update(ctx)
		if err != nil {
			writeErrorJSON(ctx, err)
			return
		}

		ctx.JSON(http.StatusNoContent, nil)
	})

	group.DELETE("", func(ctx *gin.Context) {

		err := api.Delete(ctx)
		if err != nil {
			writeErrorJSON(ctx, err)
			return
		}

		ctx.JSON(http.StatusNoContent, nil)
	})
}

func writeErrorJSON(ctx *gin.Context, err error) {

	assertedError, ok := errors.TryAssertError(err)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, CRUDResponse{Error: errors.UnknownError.New(err)})
		return
	}

	var statusCode int
	var errorResponse = CRUDResponse{Error: assertedError}

	switch assertedError.Code {
	case errors.ObjectIDNotFoundErrorCode:
		statusCode = http.StatusNotFound
	default:
		statusCode = http.StatusBadRequest
	}

	ctx.JSON(statusCode, errorResponse)
}
