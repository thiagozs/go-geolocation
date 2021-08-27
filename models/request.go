package models

type Request struct {
	Address string `form:"address" json:"address"`
}
