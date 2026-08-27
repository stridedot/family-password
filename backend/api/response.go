package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 统一响应封装：成功/失败集中处理，handler 不再直接拼 c.JSON。
// 约定：成功返回数据本体；失败统一返回 {"error": msg}。

// ok 200 + 数据
func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, data)
}

// fail 统一错误响应
func fail(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"error": msg})
}

func badRequest(c *gin.Context, msg string)      { fail(c, http.StatusBadRequest, msg) }
func notFound(c *gin.Context, msg string)        { fail(c, http.StatusNotFound, msg) }
func internal(c *gin.Context, msg string)        { fail(c, http.StatusInternalServerError, msg) }
func unprocessable(c *gin.Context, msg string)   { fail(c, http.StatusUnprocessableEntity, msg) }
