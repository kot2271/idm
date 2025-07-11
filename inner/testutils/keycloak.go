package testutils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type TestConfig struct {
	Keycloak struct {
		Realm        string `json:"Realm"`
		ClientID     string `json:"ClientID"`
		ClientSecret string `json:"ClientSecret"`
		Username1    string `json:"Username1"`
		Username2    string `json:"Username2"`
		Password     string `json:"Password"`
	} `json:"Keycloak"`
}

// KeycloakTokenResponse — структура ответа от Keycloak
type KeycloakTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

func LoadTestConfig(oneLevelUp string, twoLevelUp string) (*TestConfig, error) {
	// Получаем текущую директорию
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// Строим путь к файлу inner/testconfig.json
	path := filepath.Join(wd, oneLevelUp, twoLevelUp, "testconfig.json")

	// Читаем файл
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &TestConfig{}
	err = json.Unmarshal(file, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// GetKeycloakToken получает access_token из Keycloak
func GetKeycloakToken(ctx context.Context, realm, clientID, clientSecret, username, password string) (string, error) {
	tokenURL := fmt.Sprintf("http://localhost:9990/realms/%s/protocol/openid-connect/token", realm)

	data := fmt.Appendf(nil,
		"grant_type=password&username=%s&password=%s&client_id=%s&client_secret=%s",
		username, password, clientID, clientSecret,
	)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get token from Keycloak: %d - %s", resp.StatusCode, body)
	}

	var tokenResp KeycloakTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}

	return tokenResp.AccessToken, nil
}
