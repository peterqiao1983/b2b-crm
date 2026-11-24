package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Body struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Body{
		Code:    200,
		Data:    data,
		Message: "success",
	})
}

func Error(c *gin.Context, httpCode int, msg string) {
	c.JSON(httpCode, Body{
		Code:    httpCode,
		Data:    nil,
		Message: msg,
	})
}
