package usecase

import "github.com/hd2yao/go-training/Week02/work/domain"

type useCase struct {
    repo domain.IUserRepository
}

func NewUserUseCase(repo domain.IUserRepository) domain.IUserUseCase {
    return &useCase{repo: repo}
}

func (u *useCase) Login(userName, password string) (*domain.User, error) {
    return u.repo.Login(userName, password)
}
