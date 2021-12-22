package r

import (
	"fmt"
	"net/http"
	"qnhd/models"
	"qnhd/pkg/e"
	"qnhd/pkg/util"

	"github.com/astaxie/beego/validation"
	"github.com/fatih/structs"
	"github.com/gin-gonic/gin"
)

func GetUid(c *gin.Context) string {
	var claims *util.Claims
	token := c.GetHeader("token")
	if token == "" {
		return ""
	} else {
		claims, _ = util.ParseToken(token)
		return claims.Uid
	}
}

// 通过code和data生成一个gin.H
func H(code int, data map[string]interface{}) gin.H {
	return structs.Map(models.Response{
		Code: code,
		Msg:  e.GetMsg(code),
		Data: data,
	})
}

// 返回是否没有错误
func E(valid *validation.Validation, errorPhase string) (bool, error) {
	s := errorPhase
	if valid.HasErrors() {
		for _, r := range valid.Errors {
			s += fmt.Sprintf("\n%v %v\n", r.Key, r.Message)
		}
	}
	return !valid.HasErrors(), fmt.Errorf(s)
}

func R(c *gin.Context, httpCode int, code int, data map[string]interface{}) {
	c.JSON(httpCode, gin.H{
		"code": code,
		"msg":  e.GetMsg(code),
		"data": data,
	})
}

func Success(c *gin.Context, code int, data map[string]interface{}) {
	R(c, http.StatusOK, code, data)
}
