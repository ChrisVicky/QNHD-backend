package models

type Model struct {
	Id        uint64 `gorm:"primary_key" json:"id"`
	CreatedAt string `json:"create_at"`
	DeletedAt string `json:"delete_at" gorm:"null;"`
}

type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

type ListRes struct {
	List  interface{} `json:"list"`
	Total int         `json:"total" example:"1"`
}
