package middleware

import "github.com/gin-gonic/gin"

// V2Contract marks a request as belonging to the public v2 contract.  Legacy
// controllers still use the shared resp helpers; the marker lets those helpers
// render a resource response or RFC 9457 problem instead of the retired v1
// code/message/data envelope.
func V2Contract() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("api_version", "v2")
		c.Next()
	}
}
