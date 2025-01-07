package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/labstack/echo/v4"
)

type TestHandler struct {
	botToken string
}

func NewTestHandler(botToken string) *TestHandler {
	return &TestHandler{
		botToken: botToken,
	}
}

// @Summary Generate test query
// @Description Generate test query string for Telegram WebApp authentication
// @Tags test
// @Produce json
// @Success 200 {object} map[string]string "Generated query string"
// @Router /test [get]
func (h *TestHandler) GenerateTestQuery(c echo.Context) error {
	// Создаем тестовые данные
	queryID := "AAHdF89dhf"
	user := `{"id":123456789,"first_name":"Test","last_name":"User","username":"testuser","language_code":"en"}`
	authDate := fmt.Sprintf("%d", time.Now().Unix())

	// Создаем строку для проверки
	dataCheckString := fmt.Sprintf("auth_date=%s\nquery_id=%s\nuser=%s",
		authDate, queryID, user)

	// Создаем ключ для проверки подписи
	secretKeyGen := hmac.New(sha256.New, []byte("WebAppData"))
	secretKeyGen.Write([]byte(h.botToken))
	secretKey := secretKeyGen.Sum(nil)

	// Вычисляем HMAC-SHA-256
	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(dataCheckString))
	hash := hex.EncodeToString(mac.Sum(nil))

	// Формируем полный query string
	query := fmt.Sprintf("query_id=%s&user=%s&auth_date=%s&hash=%s",
		queryID, url.QueryEscape(user), authDate, hash)

	return c.JSON(http.StatusOK, map[string]string{
		"query":    query,
		"test_url": fmt.Sprintf("/test_post?query=%s", url.QueryEscape(query)),
	})
}

// @Summary Test query validation
// @Description Test endpoint for validating Telegram WebApp authentication
// @Tags test
// @Produce json
// @Param query query string true "Telegram WebApp authentication data"
// @Success 200 {object} map[string]string "Query validation passed"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /test_post [post]
func (h *TestHandler) ValidateTestQuery(c echo.Context) error {
	log.Printf("Успешная проверка query! Query string: %s", c.QueryString())
	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Query validation passed",
	})
}
