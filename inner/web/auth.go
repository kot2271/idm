package web

import (
	"idm/inner/common"
	"slices"

	jwtMiddleware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

const (
	JwtKey   = "jwt"
	IdmAdmin = "IDM_ADMIN"
	IdmUser  = "IDM_USER"
)

type IdmClaims struct {
	RealmAccess RealmAccessClaims `json:"realm_access"`
	jwt.RegisteredClaims
}

type RealmAccessClaims struct {
	Roles []string `json:"roles"`
}

// middleware для JWT аутентификации
func AuthMiddleware(logger *common.Logger) fiber.Handler {
	config := jwtMiddleware.Config{
		ContextKey:   JwtKey,
		ErrorHandler: createJwtErrorHandler(logger),
		JWKSetURLs:   []string{"http://localhost:9990/realms/idm/protocol/openid-connect/certs"},
		Claims:       &IdmClaims{},
	}
	return jwtMiddleware.New(config)
}

// middleware для проверки конкретной роли
func RequireRole(requiredRole string, logger *common.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Locals(JwtKey).(*jwt.Token)
		claims := token.Claims.(*IdmClaims)

		if !slices.Contains(claims.RealmAccess.Roles, requiredRole) {
			logger.Warn("Access denied: insufficient role",
				zap.String("required_role", requiredRole),
				zap.Strings("user_roles", claims.RealmAccess.Roles),
				zap.String("path", c.Path()),
				zap.String("method", c.Method()),
				zap.String("ip", c.IP()))

			return common.ErrResponse(c, fiber.StatusForbidden, "Insufficient permissions")
		}

		logger.Debug("Role check passed",
			zap.String("required_role", requiredRole),
			zap.Strings("user_roles", claims.RealmAccess.Roles),
			zap.String("path", c.Path()))

		return c.Next()
	}
}

// middleware для проверки любой из указанных ролей
func RequireAnyRole(requiredRoles []string, logger *common.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Locals(JwtKey).(*jwt.Token)
		claims := token.Claims.(*IdmClaims)

		hasRole := false
		for _, role := range requiredRoles {
			if slices.Contains(claims.RealmAccess.Roles, role) {
				hasRole = true
				break
			}
		}

		if !hasRole {
			logger.Warn("Access denied: insufficient role",
				zap.Strings("required_roles", requiredRoles),
				zap.Strings("user_roles", claims.RealmAccess.Roles),
				zap.String("path", c.Path()),
				zap.String("method", c.Method()),
				zap.String("ip", c.IP()))

			return common.ErrResponse(c, fiber.StatusForbidden, "Insufficient permissions")
		}

		logger.Debug("Role check passed",
			zap.Strings("required_roles", requiredRoles),
			zap.Strings("user_roles", claims.RealmAccess.Roles),
			zap.String("path", c.Path()))

		return c.Next()
	}
}

// извлекает роли пользователя из JWT токена
func GetUserRoles(c *fiber.Ctx) []string {
	token := c.Locals(JwtKey).(*jwt.Token)
	claims := token.Claims.(*IdmClaims)
	return claims.RealmAccess.Roles
}

// проверка, есть ли у пользователя определённая роль
func HasRole(c *fiber.Ctx, role string) bool {
	token := c.Locals(JwtKey).(*jwt.Token)
	claims := token.Claims.(*IdmClaims)
	return slices.Contains(claims.RealmAccess.Roles, role)
}

// проверка, есть ли у пользователя любая из указанных ролей
func HasAnyRole(c *fiber.Ctx, roles []string) bool {
	token := c.Locals(JwtKey).(*jwt.Token)
	claims := token.Claims.(*IdmClaims)

	for _, role := range roles {
		if slices.Contains(claims.RealmAccess.Roles, role) {
			return true
		}
	}
	return false
}

func createJwtErrorHandler(logger *common.Logger) fiber.ErrorHandler {
	return func(ctx *fiber.Ctx, err error) error {
		// Добавление X-Request-ID в заголовок ответа
		requestID := ctx.Get("X-Request-ID")
		if requestID == "" {
			requestID = ctx.Locals("requestid").(string)
		}
		ctx.Set("X-Request-ID", requestID)

		logger.ErrorCtx(ctx, "authentication failed",
			zap.Error(err),
			zap.String("path", ctx.Path()),
			zap.String("method", ctx.Method()),
			zap.String("ip", ctx.IP()))

		// Если токен не может быть прочитан, то возвращаем 401
		return common.ErrResponse(
			ctx,
			fiber.StatusUnauthorized,
			err.Error(),
		)
	}
}
