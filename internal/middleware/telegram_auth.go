package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/labstack/echo/v4"
)

type TelegramAuthConfig struct {
	BotToken string
}

func TelegramAuthMiddleware(config TelegramAuthConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Проверяем только POST запросы
			if c.Request().Method != http.MethodPost {
				return next(c)
			}

			// Получаем query параметры
			query := c.QueryString()
			if query == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Missing Telegram WebApp authentication data",
				})
			}

			// Проверяем подпись
			if !checkTelegramAuth(query, config.BotToken) {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Invalid Telegram WebApp authentication",
				})
			}

			return next(c)
		}
	}
}

func checkTelegramAuth(query, botToken string) bool {
	// Создаем ключ для проверки подписи
	secretKeyGen := hmac.New(sha256.New, []byte("WebAppData"))
	secretKeyGen.Write([]byte(botToken))
	secretKey := secretKeyGen.Sum(nil)

	// Парсим query параметры
	parsedData, err := url.ParseQuery(query)
	if err != nil {
		return false
	}

	// Проверяем наличие обязательных полей
	requiredKeys := []string{"user", "auth_date", "hash"}
	for _, key := range requiredKeys {
		if _, ok := parsedData[key]; !ok {
			return false
		}
	}

	// Создаем отсортированную строку для проверки
	var keys []string
	for key := range parsedData {
		if key != "hash" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	var dataCheckStrings []string
	for _, key := range keys {
		dataCheckStrings = append(dataCheckStrings, fmt.Sprintf("%s=%s", key, parsedData.Get(key)))
	}
	dataCheckString := strings.Join(dataCheckStrings, "\n")

	// Вычисляем HMAC-SHA-256
	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(dataCheckString))
	calculatedHash := hex.EncodeToString(mac.Sum(nil))

	return calculatedHash == parsedData.Get("hash")
}
