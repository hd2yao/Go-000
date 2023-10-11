package repository

import (
    "github.com/pkg/errors"
    "gorm.io/gorm"

    "github.com/hd2yao/go-training/Week02/work/domain"
    "github.com/hd2yao/go-training/Week02/work/pkg/errcode"
)

type repository struct {
    db *gorm.DB
}

func NewUserRepository(db *gorm.DB) domain.IUserRepository {
    return &repository{db: db}
}

func (r *repository) Login(userName, password string) (*domain.User, error) {
    var user domain.User
    err := r.db.Where(domain.User{Name: userName, Password: password}).First(&user).Error
    if err == gorm.ErrRecordNotFound {
        return nil, errors.Wrap(errcode.UserLogin, "ErrRecordNotFound")
    }
    if err != nil {
        return nil, errors.Wrapf(errcode.DBQuery, "db errors id: %+v", err)
    }
    return &user, nil
}
