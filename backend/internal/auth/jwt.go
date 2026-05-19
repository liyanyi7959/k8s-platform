// auth 提供认证相关能力（当前主要是 JWT 的签发与解析）。
//
// 约定：
// - 使用 HS256 对称签名（secret 在服务端保存）
// - 令牌有效期由配置控制（例如 168h），过期后必须重新登录
// - Claims 中会携带角色与权限点快照，便于无 DB 的权限判断；如需实时权限可在中间件中刷新
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	// UserID 为登录用户的唯一标识（由后端数据库用户表决定）。
	UserID int64 `json:"user_id"`
	// Username 为登录用户名，用于前端展示与审计定位。
	Username string `json:"username"`
	// Roles 为用户拥有的角色名称列表（例如 ["admin"]）。
	// 说明：当前后端在签发 token 时把角色快照写入 token，便于无 DB 访问的权限判断。
	Roles []string `json:"roles,omitempty"`
	// Perms 为用户拥有的权限点列表（例如 ["cluster:read","k8s:read"]）。
	// 说明：同样是“快照”设计；当后台修改角色权限后，旧 token 直到过期前可能仍携带旧权限点。
	Perms []string `json:"perms,omitempty"`
	jwt.RegisteredClaims
}

type Manager struct {
	// secret 为 HS256 的签名密钥。
	// 生产环境必须通过配置覆盖，避免默认值带来安全风险。
	secret []byte
}

func NewManager(secret string) *Manager {
	// 若未配置 secret，则使用开发默认值，确保本地可运行。
	if secret == "" {
		secret = "dev_secret_change_me"
	}
	return &Manager{secret: []byte(secret)}
}

func (m *Manager) IssueToken(params Claims, ttl time.Duration) (string, error) {
	// IssueToken 用于签发 JWT。ttl 由配置决定（例如 168h）。
	now := time.Now()
	claims := params
	claims.RegisteredClaims = jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(m.secret)
}

func (m *Manager) ParseToken(tokenString string) (*Claims, error) {
	// ParseToken 用于解析并校验 JWT（算法、签名、过期时间等）。
	// 成功时返回 Claims 指针，供中间件写入 gin.Context。
	tok, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		// 防止算法降级攻击：只接受 HS256。
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !tok.Valid {
		return nil, errors.New("invalid token")
	}
	claims, ok := tok.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	return claims, nil
}
