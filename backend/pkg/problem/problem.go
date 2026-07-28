package problem

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Details struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Detail    string `json:"detail,omitempty"`
	Instance  string `json:"instance,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

func Write(c *gin.Context, status int, problemType, title, detail string) {
	requestID, _ := c.Get("request_id")
	rid, _ := requestID.(string)
	c.Set("resp_code", status)
	c.Set("resp_message", title)
	c.Header("Content-Type", "application/problem+json")
	c.JSON(status, Details{Type: problemType, Title: title, Status: status, Detail: detail, Instance: c.Request.URL.Path, RequestID: rid})
}

func Unauthorized(c *gin.Context, detail string) {
	Write(c, http.StatusUnauthorized, "https://aiops.local/problems/unauthorized", "认证失败", detail)
}

func Forbidden(c *gin.Context, detail string) {
	Write(c, http.StatusForbidden, "https://aiops.local/problems/forbidden", "权限不足", detail)
}
