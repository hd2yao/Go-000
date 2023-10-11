package domain

import "gorm.io/gorm"

// User user
type User struct {
    gorm.Model

    Name     string `json:"name"`
    Password string `json:"password"`
}

type IUserRepository interface {
    Login(userName, password string) (*User, error)
}

type IUserUsecase interface {
    Login(userName, password string) (*User, error)
}
