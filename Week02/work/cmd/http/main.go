package main

import (
    "log"

    "github.com/gin-gonic/gin"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"

    "github.com/hd2yao/go-training/Week02/work/domain"
    "github.com/hd2yao/go-training/Week02/work/user/api"
    "github.com/hd2yao/go-training/Week02/work/user/repository"
    "github.com/hd2yao/go-training/Week02/work/user/usecase"
)

func main() {
    db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
    if err != nil {
        log.Panicf("db is must: %+v", err)
    }

    db.AutoMigrate(&domain.User{})

    e := gin.Default()

    // 后面可以用 wire 进行依赖注入
    repo := repository.NewUserRepository(db)
    useCase := usecase.NewUserUseCase(repo)
    api.NewUserHandler(e, useCase)

    if err := e.Run(":8080"); err != nil {
        log.Fatalf("gin exit: %+v", err)
    }
}
