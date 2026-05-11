package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func Cache() func(c *gin.Context) {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/" {
			c.Header("Cache-Control", "no-cache")
		} else if isAPILikePath(path) {
			c.Header("Cache-Control", "no-store, no-cache, must-revalidate, private, max-age=0")
		} else {
			c.Header("Cache-Control", "max-age=604800") // one week
		}
		c.Header("Cache-Version", "b688f2fb5be447c25e5aa3bd063087a83db32a288bf6a4f35f2d8db310e40b14")
		c.Next()
	}
}

func isAPILikePath(path string) bool {
	prefixes := []string{"/api", "/v1", "/v1beta", "/mj", "/pg", "/suno"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
