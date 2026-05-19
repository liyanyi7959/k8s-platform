package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/service"
)

type cacheBodyWriter struct {
	gin.ResponseWriter
	buf bytes.Buffer
}

func (w *cacheBodyWriter) Write(b []byte) (int, error) {
	_, _ = w.buf.Write(b)
	return w.ResponseWriter.Write(b)
}

func CacheJSON(store service.CacheStore, ttl time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if store == nil || !store.Enabled() || c.Request.Method != http.MethodGet {
			c.Next()
			return
		}

		c.Header("X-Cache", "MISS")
		uid := int64(0)
		if claims, ok := GetClaims(c); ok && claims != nil {
			uid = claims.UserID
		}
		rawKey := "uid=" + itoa64(uid) + "|m=" + c.Request.Method + "|p=" + c.Request.URL.Path + "|q=" + c.Request.URL.RawQuery
		sum := sha256.Sum256([]byte(rawKey))
		key := "httpcache:v1:" + hex.EncodeToString(sum[:])

		rctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
		b, ok, err := store.Get(rctx, key)
		cancel()
		if err != nil {
			c.Header("X-Cache", "BYPASS")
		} else if ok && len(b) > 0 {
			c.Header("X-Cache", "HIT")
			c.Data(http.StatusOK, "application/json; charset=utf-8", b)
			c.Abort()
			return
		}

		w := &cacheBodyWriter{ResponseWriter: c.Writer}
		c.Writer = w
		c.Next()

		if c.IsAborted() {
			return
		}
		if c.Writer.Status() != http.StatusOK {
			return
		}
		body := w.buf.Bytes()
		if len(body) == 0 {
			return
		}

		var r struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(body, &r); err != nil || r.Code != 0 {
			return
		}

		wctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		_ = store.Set(wctx, key, body, ttl)
		cancel()
	}
}

func itoa64(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [32]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
