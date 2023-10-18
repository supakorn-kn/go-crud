package objects

import "reflect"

type User struct {
	UserID      string `json:"user_id" bson:"user_id,omitempty"`
	Username    string `json:"username" bson:"username,omitempty"`
	Password    string `json:"password" bson:"password,omitempty"`
	AccountName string `json:"account_name" bson:"account_name,omitempty"`
	Email       string `json:"email" bson:"email,omitempty"`
}

func (u User) GetID() string {
	return u.UserID
}

func (u User) IsNil() bool {
	return reflect.ValueOf(u).IsZero()
}
