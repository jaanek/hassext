package model

type LoginUP struct {
	User
	ID       uint   `json:"id" db:"id"`
	Username string `json:"username" db:"username"`
	Password string `json:"-"`
}

func (u *LoginUP) TableName() string {
	return "loginup"
}
