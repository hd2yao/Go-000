package api

import (
    "github.com/gin-gonic/gin"
    "github.com/pkg/errors"
    "net/http"

    "github.com/hd2yao/go-training/Week02/work/domain"
    "github.com/hd2yao/go-training/Week02/work/pkg/errcode"
)

type api struct {
    useCase domain.IUserUseCase
    r       gin.IRouter
}

func NewUserHandler(r gin.IRouter, userCase domain.IUserUseCase) {
    api := &api{useCase: userCase}
    r.POST("/login", api.Login)
}

type loginParam struct {
    UserName string `json:"user_name"`
    Password string `json:"password"`
}

func (a *api) Login(context *gin.Context) {
    var params loginParam
    if err := context.ShouldBind(&params); err != nil {
        errResp(context, errors.Wrapf(errcode.ErrParams, "should bind err: %+v", err))
        return
    }
    user, err := a.useCase.Login(params.UserName, params.Password)
    if err != nil {
        errResp(context, err)
        return
    }
    context.JSON(http.StatusOK, gin.H{"code": "success", "message": "登录成功", "data": user.ID})
}

func errResp(context *gin.Context, err error) {
    var code *errcode.ErrorCode
    if !errors.As(err, &code) {
        code = errcode.ErrUnKnown
    }
    context.Error(err)
    context.JSON(code.Status, code)
}
