package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const bearerPrefix = "Bearer"

// CORSMiddleware разрешает запросы из браузера и Postman/Insomnia (как в учебном примере).
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func jwtKeyBytes() []byte {
	k := os.Getenv("JWT_KEY")
	if k == "" {
		k = "default-secret-key-change-in-production"
	}
	return []byte(k)
}

// JWTAuthMiddleware проверяет JWT и blacklist в Redis. requireModerator=true — только модератор (403 иначе).
func (h *Handler) JWTAuthMiddleware(requireModerator bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractTokenFromHeader(c.Request)
		if tokenString == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtKeyBytes(), nil
		})
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		blacklisted, err := h.Repository.IsTokenBlacklisted(context.Background(), tokenString)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if blacklisted {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		userIDStr, ok := claims["user_id"].(string)
		if !ok {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		jwtIsModerator, ok := claims["is_moderator"].(bool)
		if !ok {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		if requireModerator && !jwtIsModerator {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		c.Set("user_id", userIDStr)
		c.Set("is_moderator", jwtIsModerator)
		c.Next()
	}
}

// WithOptionalAuthCheck подставляет user_id в контекст, если передан валидный незаблокированный JWT.
func (h *Handler) WithOptionalAuthCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if tokenString == "" || !strings.HasPrefix(tokenString, prefix) {
			c.Set("user_id", "")
			c.Next()
			return
		}
		tokenString = strings.TrimPrefix(tokenString, prefix)

		blacklisted, err := h.Repository.IsTokenBlacklisted(context.Background(), tokenString)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if blacklisted {
			c.Set("user_id", "")
			c.Next()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtKeyBytes(), nil
		})
		if err != nil || token == nil || !token.Valid {
			c.Set("user_id", "")
			c.Next()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.Set("user_id", "")
			c.Next()
			return
		}

		userIDValue, exists := claims["user_id"]
		if !exists || userIDValue == nil {
			c.Set("user_id", "")
			c.Next()
			return
		}

		var userIDStr string
		switch v := userIDValue.(type) {
		case string:
			userIDStr = v
		case float64:
			userIDStr = fmt.Sprintf("%.0f", v)
		default:
			userIDStr = fmt.Sprintf("%v", v)
		}

		c.Set("user_id", userIDStr)
		c.Next()
	}
}

func extractTokenFromHeader(r *http.Request) string {
	bearerToken := r.Header.Get("Authorization")
	if bearerToken == "" {
		return ""
	}
	parts := strings.SplitN(bearerToken, " ", 2)
	if len(parts) != 2 || parts[0] != bearerPrefix {
		return ""
	}
	return parts[1]
}

// getUserID из контекста Gin после JWTAuthMiddleware.
func getUserID(ctx *gin.Context) (int, error) {
	userIDVal, exists := ctx.Get("user_id")
	if !exists || userIDVal == nil {
		return 0, fmt.Errorf("user_id not found")
	}
	userIDStr, ok := userIDVal.(string)
	if !ok || userIDStr == "" {
		return 0, fmt.Errorf("user_id not found")
	}
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return 0, err
	}
	return userID, nil
}

func isModeratorFromContext(ctx *gin.Context) bool {
	v, ok := ctx.Get("is_moderator")
	if !ok {
		return false
	}
	b, _ := v.(bool)
	return b
}

func tokenTTLFromClaims(claims jwt.MapClaims) (time.Duration, error) {
	expVal, ok := claims["exp"]
	if !ok {
		return 0, errors.New("exp not present")
	}

	var expUnix int64
	switch v := expVal.(type) {
	case float64:
		expUnix = int64(v)
	case int64:
		expUnix = v
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return 0, err
		}
		expUnix = i
	default:
		return 0, fmt.Errorf("unsupported exp type %T", v)
	}

	expTime := time.Unix(expUnix, 0)
	ttl := time.Until(expTime)
	if ttl < 0 {
		return 0, errors.New("token already expired")
	}
	return ttl, nil
}
