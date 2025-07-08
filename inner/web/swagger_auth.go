package web

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
)

// возвращает конфигурацию OAuth2.0 для Swagger UI
func GetSwaggerOAuthConfig() *swagger.OAuthConfig {
	return &swagger.OAuthConfig{
		// Название приложения, отображаемое в Swagger UI
		AppName: "IDM API",

		// Client ID для Swagger UI (должен быть настроен в Keycloak)
		ClientId: "idm-swagger-ui",

		// Realm в Keycloak
		Realm: "idm",

		// Дополнительные параметры для OAuth2.0
		AdditionalQueryStringParams: map[string]string{
			"nonce": "defaultNonce",
		},

		// Scopes, которые будут запрошены
		Scopes: []string{
			"openid",
			"profile",
			"email",
			"read",
			"write",
		},

		// Использовать PKCE для Authorization Code flow
		UsePkceWithAuthorizationCodeGrant: true,
	}
}

// возвращает полную конфигурацию Swagger с OAuth2.0
func GetSwaggerConfig() swagger.Config {
	return swagger.Config{
		// URL для получения OpenAPI спецификации
		URL: "/swagger/doc.json",

		// Включить deep linking
		DeepLinking: true,

		// Настройки раскрытия разделов по умолчанию
		DocExpansion: "none",

		// Включить валидацию запросов
		ValidatorUrl: "",

		// Конфигурация OAuth2.0
		OAuth: GetSwaggerOAuthConfig(),

		// Дополнительные настройки UI
		DefaultModelsExpandDepth: 1,
		DefaultModelExpandDepth:  1,
		DefaultModelRendering:    "model",

		// Показывать панель авторизации
		ShowExtensions:       true,
		ShowCommonExtensions: true,

		// Настройки для работы с CORS
		SupportedSubmitMethods: []string{
			"get", "post", "put", "delete", "patch",
		},

		// Настройки интерфейса
		Layout: "StandaloneLayout",

		// Кастомный CSS (опционально)
		CustomStyle: `
			.swagger-ui .topbar { 
				display: none; 
			}
			.swagger-ui .scheme-container { 
				background: #1f2937;
				padding: 10px;
				border-radius: 5px;
			}
		`,

		// Заголовок страницы
		Title: "IDM API Documentation",
	}
}

// Константы для OAuth2.0 endpoints (для использования в других частях приложения)
const (
	// Keycloak endpoints
	KeycloakBaseURL      = "http://localhost:9990"
	KeycloakRealm        = "idm"
	KeycloakClientID     = "idm-swagger-ui"
	KeycloakClientSecret = "" // Для public client может быть пустым

	// OAuth2.0 endpoints
	TokenURL         = KeycloakBaseURL + "/realms/" + KeycloakRealm + "/protocol/openid-connect/token"
	AuthorizationURL = KeycloakBaseURL + "/realms/" + KeycloakRealm + "/protocol/openid-connect/auth"
	UserInfoURL      = KeycloakBaseURL + "/realms/" + KeycloakRealm + "/protocol/openid-connect/userinfo"
	JWKSUrl          = KeycloakBaseURL + "/realms/" + KeycloakRealm + "/protocol/openid-connect/certs"

	// Scopes
	ScopeOpenID  = "openid"
	ScopeProfile = "profile"
	ScopeEmail   = "email"
	ScopeRead    = "read"
	ScopeWrite   = "write"
)

// возвращает список всех доступных scopes
func OAuthScopes() []string {
	return []string{
		ScopeOpenID,
		ScopeProfile,
		ScopeEmail,
		ScopeRead,
		ScopeWrite,
	}
}

// инициализирует Swagger UI с поддержкой OAuth2.0
func InitSwaggerWithOAuth(app *fiber.App) {
	config := GetSwaggerConfig()
	app.Use("/swagger/*", swagger.New(config))

	// Дополнительный endpoint для получения конфигурации OAuth2.0
	app.Get("/swagger/oauth2-redirect.html", func(c *fiber.Ctx) error {
		return c.SendString(`
<!DOCTYPE html>
<html lang="en-US">
<head>
    <title>Swagger UI: OAuth2 Redirect</title>
</head>
<body>
<script>
    'use strict';
    function run () {
        var oauth2 = window.opener.swaggerUIRedirectOauth2;
        var sentState = oauth2.state;
        var redirectUrl = oauth2.redirectUrl;
        var isValid, qp, arr;

        if (/code|token|error/.test(window.location.hash)) {
            qp = window.location.hash.substring(1);
        } else {
            qp = location.search.substring(1);
        }

        arr = qp.split("&");
        arr.forEach(function (v,i,_arr) { _arr[i] = '"' + v.replace('=', '":"') + '"';});
        qp = qp ? JSON.parse('{' + arr.join(',') + '}', function (key, value) {
            return key === "" ? value : decodeURIComponent(value);
        }) : {};

        isValid = qp.state === sentState;

        if ((
          oauth2.auth.schema.get("flow") === "accessCode" ||
          oauth2.auth.schema.get("flow") === "authorizationCode" ||
          oauth2.auth.schema.get("flow") === "authorization_code"
        ) && !oauth2.auth.code) {
            if (!isValid) {
                oauth2.errCb({
                    authId: oauth2.auth.name,
                    source: "auth",
                    level: "warning",
                    message: "Authorization may be unsafe, passed state was changed in server Passed state wasn't returned from auth server"
                });
            }

            if (qp.code) {
                delete oauth2.state;
                oauth2.auth.code = qp.code;
                oauth2.callback({auth: oauth2.auth, redirectUrl: redirectUrl});
            } else {
                let oauthErrorMsg;
                if (qp.error) {
                    oauthErrorMsg = "["+qp.error+"]: " +
                        (qp.error_description ? qp.error_description+ ". " : "no accessCode received from the server. ") +
                        (qp.error_uri ? "More info: "+qp.error_uri : "");
                }

                oauth2.errCb({
                    authId: oauth2.auth.name,
                    source: "auth",
                    level: "error",
                    message: oauthErrorMsg || "[Authorization failed]: no accessCode received from the server"
                });
            }
        } else {
            oauth2.callback({auth: oauth2.auth, token: qp, isValid: isValid, redirectUrl: redirectUrl});
        }
        window.close();
    }

    window.addEventListener('DOMContentLoaded', function () {
        run();
    });
</script>
</body>
</html>
		`)
	})
}
