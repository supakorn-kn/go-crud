package users

import (
	"github.com/gin-gonic/gin"
	"github.com/supakorn-kn/go-crud/models"
	"github.com/supakorn-kn/go-crud/models/users"
	"github.com/supakorn-kn/go-crud/mongodb"
	"github.com/supakorn-kn/go-crud/objects"
)

type UsersCrudAPI struct {
	model users.UsersModel
}

func NewUsersAPI(conn *mongodb.MongoDBConn) (*UsersCrudAPI, error) {

	model, err := users.NewUsersModel(conn)
	if err != nil {
		return nil, err
	}

	api := new(UsersCrudAPI)
	api.model = *model

	return api, nil
}

func (api UsersCrudAPI) Insert(ctx *gin.Context) error {

	var user objects.User
	err := ctx.BindJSON(&user)
	if err != nil {
		return err
	}

	err = api.model.Insert(user)
	if err != nil {
		return err
	}

	return nil
}

func (api UsersCrudAPI) ReadOne(itemID string, ctx *gin.Context) (*objects.User, error) {

	user, err := api.model.GetByID(itemID)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (api UsersCrudAPI) Read(ctx *gin.Context) (*models.PaginationData[objects.User], error) {

	var opt users.SearchOptions
	err := ctx.BindJSON(&opt)
	if err != nil {
		return nil, err
	}

	paginationData, err := api.model.Search(opt)
	if err != nil {
		return nil, err
	}

	return &paginationData, nil
}

func (api UsersCrudAPI) Update(ctx *gin.Context) error {

	var user objects.User
	err := ctx.BindJSON(&user)
	if err != nil {
		return err
	}

	err = api.model.Update(user)
	if err != nil {
		return err
	}

	return nil
}

func (api UsersCrudAPI) Delete(ctx *gin.Context) error {

	var user objects.User
	err := ctx.BindJSON(&user)
	if err != nil {
		return err
	}

	err = api.model.Delete(user.UserID)
	if err != nil {
		return err
	}

	return nil
}
