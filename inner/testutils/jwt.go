package testutils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("secret")

func GenerateExpiredToken() string {
	claims := jwt.MapClaims{
		"sub":  "1234567890",
		"name": "Test User",
		"iat":  time.Now().Add(-2 * time.Hour).Unix(), // создан 2 часа назад
		"exp":  time.Now().Add(-1 * time.Hour).Unix(), // истёк 1 час назад
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, _ := token.SignedString(jwtSecret)
	return signedToken
}

func GenerateRolelessToken() string {
	claims := jwt.MapClaims{
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
		"sub":   "1234567890",
		"typ":   "Bearer",
		"azp":   "idmapp",
		"name":  "Test User",
		"email": "test-user@test.com",
		"iss":   "http://localhost:9990/realms/idm",
		"roles": "offline_access",
	}

	cfg, _ := LoadTestConfig("..", "")
	secret := cfg.Keycloak.ClientSecret

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, _ := token.SignedString([]byte(secret))
	return signedToken
}
