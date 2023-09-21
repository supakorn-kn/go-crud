package apis

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterCrudAPI[Item any](pathName string, api CrudAPI[Item], g *gin.Engine) {

	g.POST(pathName, func(ctx *gin.Context) {

		err := api.Insert(ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, err)
			return
		}

		ctx.JSON(http.StatusCreated, OKResponse)
	})

	g.GET(pathName+"/:id", func(ctx *gin.Context) {

		itemID := ctx.Param("id")
		item, err := api.ReadOne(itemID, ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, err)
			return
		}

		ctx.JSON(http.StatusOK, item)
	})

	g.GET(pathName, func(ctx *gin.Context) {

		paginateResult, err := api.Read(ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, err)
			return
		}

		ctx.JSON(http.StatusOK, paginateResult)
	})

	g.PUT(pathName, func(ctx *gin.Context) {

		err := api.Update(ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, err)
			return
		}

		ctx.JSON(http.StatusOK, OKResponse)
	})

	g.DELETE(pathName, func(ctx *gin.Context) {

		err := api.Delete(ctx)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, err)
			return
		}

		ctx.JSON(http.StatusOK, OKResponse)
	})
}
