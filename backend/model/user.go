package model

type User struct {
	Name  string `json:"name" db:"name"`
	Email string `json:"email" db:"email"`
}

func (u *User) TableName() string {
	return "user"
}
