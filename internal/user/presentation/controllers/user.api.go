package controllers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/Peranum/tg-dice/internal/user/application/services"
	"github.com/Peranum/tg-dice/internal/user/domain/entities"
	"github.com/labstack/echo/v4"
)

type UserController struct {
	UserAppService *services.UserAppService
}

func NewUserController(userAppService *services.UserAppService) *UserController {
	return &UserController{
		UserAppService: userAppService,
	}
}

type Withdrawal struct {
	ID         string  `json:"id"`
	Amount     float64 `json:"amount"`
	Wallet     string  `json:"wallet"`
	JettonName string  `json:"jetton_name"`
	CreatedAt  string  `json:"created_at"`
}

// CreateUser handles POST /users
// @Summary Create a new user
// @Description Для создания Необходим только валлет
// @Tags users
// @Accept json
// @Produce json
// @Param user body entities.User true "User details"
// @Success 201 {object} entities.User
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users [post]
func (uc *UserController) CreateUser(c echo.Context) error {
	var user entities.User
	if err := c.Bind(&user); err != nil {
		log.Printf("[CreateUser] Error binding input: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input"})
	}

	log.Printf("[CreateUser] Input data: %+v", user)
	createdUser, err := uc.UserAppService.CreateUser(c.Request().Context(), &user)
	if err != nil {
		log.Printf("[CreateUser] Error creating user: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	log.Printf("[CreateUser] User created successfully: %+v", createdUser)
	return c.JSON(http.StatusCreated, createdUser)
}

// GetUser handles GET /users/:id
// @Summary Get a user by ID
// @Description Retrieve a user by their unique ID
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} entities.User
// @Failure 404 {object} map[string]string
// @Router /users/{id} [get]
func (uc *UserController) GetUser(c echo.Context) error {
	id := c.Param("id")

	user, err := uc.UserAppService.GetUser(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}

	return c.JSON(http.StatusOK, user)
}

// PatchUserByTgID handles PATCH /users/tgid/:tgid
// @Summary Partially update a user by TgID
// @Description Update only specific fields of a user identified by TgID
// @Tags users
// @Accept json
// @Produce json
// @Param tgid path string true "Telegram ID"
// @Param updateData body map[string]interface{} true "Fields to update"
// @Success 200 {object} map[string]interface{} "Updated fields"
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users/tgid/{tgid} [patch]
func (uc *UserController) PatchUserByTgID(c echo.Context) error {
	tgid := c.Param("tgid")
	updateData := make(map[string]interface{})

	if err := c.Bind(&updateData); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input"})
	}

	if len(updateData) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "No fields provided for update"})
	}

	err := uc.UserAppService.PatchUserByTgID(c.Request().Context(), tgid, updateData)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "User updated successfully",
		"fields":  updateData,
	})
}

// DeleteUser handles DELETE /users/:id
// @Summary Delete a user by ID
// @Description Delete a user from the system by their ID
// @Tags users
// @Param id path string true "User ID"
// @Success 204
// @Failure 500 {object} map[string]string
// @Router /users/{id} [delete]
func (uc *UserController) DeleteUser(c echo.Context) error {
	id := c.Param("id")

	err := uc.UserAppService.DeleteUser(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusNoContent, nil)
}

// ListUsers handles GET /users
// @Summary List users with pagination
// @Description Retrieve a list of users with optional pagination
// @Tags users
// @Produce json
// @Param limit query int false "Number of users to retrieve"
// @Param offset query int false "Offset for pagination"
// @Success 200 {array} entities.User
// @Failure 500 {object} map[string]string
// @Router /users [get]
func (uc *UserController) ListUsers(c echo.Context) error {
	limit := int64(10)
	offset := int64(0)

	if l := c.QueryParam("limit"); l != "" {
		parsedLimit, err := strconv.ParseInt(l, 10, 64)
		if err == nil {
			limit = parsedLimit
		}
	}
	if o := c.QueryParam("offset"); o != "" {
		parsedOffset, err := strconv.ParseInt(o, 10, 64)
		if err == nil {
			offset = parsedOffset
		}
	}

	users, err := uc.UserAppService.ListUsers(c.Request().Context(), limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, users)
}

// UpdateTokensRequest представляет запрос для обновления балансов токенов
type UpdateTokensRequest struct {
	TonBalance *float64 `json:"ton_balance,omitempty"`
	M5Balance  *float64 `json:"m5_balance,omitempty"`
	DfcBalance *float64 `json:"dfc_balance,omitempty"`
}

// UpdateUserTokens handles PATCH /users/{wallet}/tokens
// @Summary Update user token balances
// @Description Update the balances of one or more tokens for a user by their wallet
// @Tags users
// @Accept json
// @Produce json
// @Param wallet path string true "User Wallet"
// @Param body body UpdateTokensRequest true "Token balances to update (e.g., {\"ton_balance\": 10, \"m5_balance\": -5})"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users/{wallet}/tokens [patch]
func (uc *UserController) UpdateUserTokens(c echo.Context) error {
	wallet := c.Param("wallet")

	// Читаем тело запроса
	var tokenUpdates map[string]float64
	if err := c.Bind(&tokenUpdates); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input"})
	}

	// Извлекаем значения для токенов (по умолчанию nil)
	var tonBalance, m5Balance, dfcBalance *float64

	if val, exists := tokenUpdates["ton_balance"]; exists {
		tonBalance = &val
	}
	if val, exists := tokenUpdates["m5_balance"]; exists {
		m5Balance = &val
	}
	if val, exists := tokenUpdates["dfc_balance"]; exists {
		dfcBalance = &val
	}

	// Вызываем метод из аппликационного сервиса, передавая wallet
	err := uc.UserAppService.UpdateUserTokens(c.Request().Context(), wallet, tonBalance, m5Balance, dfcBalance)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Token balances updated successfully"})
}

type AddCubesRequest struct {
	CubesToAdd int `json:"cubes_to_add" example:"10"`
}

// AddCubes handles PATCH /users/{wallet}/cubes
// @Summary Add cubes to a user
// @Description Increment the number of cubes for a user by their wallet
// @Tags users
// @Accept json
// @Produce json
// @Param wallet path string true "User Wallet"
// @Param body body AddCubesRequest true "Number of cubes to add"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users/{wallet}/cubes [patch]
func (uc *UserController) AddCubes(c echo.Context) error {
	wallet := c.Param("wallet")

	// Читаем тело запроса
	var requestData struct {
		CubesToAdd int `json:"cubes_to_add"`
	}
	if err := c.Bind(&requestData); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input"})
	}

	// Проверяем, что количество кубов корректное
	if requestData.CubesToAdd <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Cubes must be greater than zero"})
	}

	// Вызываем метод аппликационного сервиса
	err := uc.UserAppService.AddCubes(c.Request().Context(), wallet, requestData.CubesToAdd)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Cubes added successfully"})
}

// GetUserBalances handles GET /users/{wallet}/balances
// @Summary Get user balances
// @Description Get token balances (ton_balance, m5_balance, dfc_balance) and cubes for a user by their wallet
// @Tags users
// @Produce json
// @Param wallet path string true "User Wallet"
// @Success 200 {object} map[string]interface{} "Balances (ton_balance, m5_balance, dfc_balance, cubes)"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /users/{wallet}/balances [get]
func (uc *UserController) GetUserBalances(c echo.Context) error {
	wallet := c.Param("wallet")

	balances, err := uc.UserAppService.GetUserBalances(c.Request().Context(), wallet)
	if err != nil {
		if err.Error() == "user not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, balances)
}

// GetReferralCodeHandler handles GET /users/{wallet}/referral-code
// @Summary Get referral code by user wallet
// @Description Retrieve the referral code for a user by their wallet
// @Tags users
// @Accept json
// @Produce json
// @Param wallet path string true "User Wallet"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users/{wallet}/referral-code [get]
func (uc *UserController) GetReferralCodeHandler(c echo.Context) error {
	// Извлекаем кошелек из параметров пути
	wallet := c.Param("wallet")
	if wallet == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Wallet is required"})
	}

	// Вызов App-сервиса для получения реферального кода
	referralCode, err := uc.UserAppService.GetReferralCodeByWallet(c.Request().Context(), wallet)
	if err != nil {
		if err.Error() == "user not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Возврат успешного ответа с реферальным кодом
	return c.JSON(http.StatusOK, map[string]string{"referral_code": referralCode})
}

// GetReferralEarnings handles GET /users/{wallet}/referral-earnings
// @Summary Get referral earnings by user wallet
// @Description Retrieve all referral earnings (ton_balance, m5_balance, dfc_balance) for a user by their wallet
// @Tags users
// @Produce json
// @Param wallet path string true "User Wallet"
// @Success 200 {object} map[string]float64 "Referral earnings (ton_balance, m5_balance, dfc_balance)"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users/{wallet}/referral-earnings [get]
func (uc *UserController) GetReferralEarnings(c echo.Context) error {
	wallet := c.Param("wallet")

	if wallet == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Wallet is required"})
	}

	// Вызываем сервис для получения реферальных доходов
	referralEarnings, err := uc.UserAppService.GetReferralEarnings(c.Request().Context(), wallet)
	if err != nil {
		if err.Error() == "user not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Возвращаем реферальные доходы
	return c.JSON(http.StatusOK, referralEarnings)
}

// GetUserByName handles GET /users/name/:name
// @Summary Get a user by name
// @Description Retrieve a user by their unique name
// @Tags users
// @Produce json
// @Param name path string true "User Name"
// @Success 200 {object} entities.User
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users/name/{name} [get]
func (uc *UserController) GetUserByName(c echo.Context) error {
	name := c.Param("name")

	user, err := uc.UserAppService.GetUserByName(c.Request().Context(), name)
	if err != nil {
		if err.Error() == "user not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, user)
}

// GetUserPointsByWallet handles GET /users/{wallet}/points
// @Summary Get user points by wallet
// @Description Retrieve the points of a user by their wallet
// @Tags users
// @Produce json
// @Param wallet path string true "User Wallet"
// @Success 200 {object} map[string]float64 "User Points"
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users/{wallet}/points [get]
func (uc *UserController) GetUserPointsByWallet(c echo.Context) error {
	wallet := c.Param("wallet")

	points, err := uc.UserAppService.GetUserPointsByWallet(c.Request().Context(), wallet)
	if err != nil {
		if err.Error() == "user not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]float64{"points": points})
}

// GetUsersSortedByPoints handles GET /users/points
// @Summary Get users sorted by points
// @Description Retrieve a list of users sorted by their points in descending order
// @Tags users
// @Produce json
// @Param limit query int false "Number of users to retrieve"
// @Param offset query int false "Offset for pagination"
// @Success 200 {array} entities.User
// @Failure 500 {object} map[string]string
// @Router /users/points [get]
func (uc *UserController) GetUsersSortedByPoints(c echo.Context) error {
	limit := int64(10)
	offset := int64(0)

	if l := c.QueryParam("limit"); l != "" {
		parsedLimit, err := strconv.ParseInt(l, 10, 64)
		if err == nil {
			limit = parsedLimit
		}
	}
	if o := c.QueryParam("offset"); o != "" {
		parsedOffset, err := strconv.ParseInt(o, 10, 64)
		if err == nil {
			offset = parsedOffset
		}
	}

	users, err := uc.UserAppService.GetUsersSortedByPoints(c.Request().Context(), limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, users)
}

type CreateWithdrawalRequest struct {
	Amount     float64 `json:"amount"`
	Wallet     string  `json:"wallet"`
	JettonName *string `json:"jetton_name,omitempty"`
}

// CreateWithdrawal handles POST /withdrawals
// @Summary Create a new withdrawal
// @Description Создание нового запроса на вывод средств
// @Tags withdrawals
// @Accept json
// @Produce json
// @Param withdrawal body CreateWithdrawalRequest true "Withdrawal details"
// @Success 201 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /withdrawals [post]
func (uc *UserController) CreateWithdrawal(c echo.Context) error {
	var request CreateWithdrawalRequest

	// Bind the incoming JSON body to the CreateWithdrawalRequest struct
	if err := c.Bind(&request); err != nil {
		log.Printf("[CreateWithdrawal] Error binding input: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input"})
	}

	// Validate the withdrawal amount
	if request.Amount <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Withdrawal amount must be greater than zero"})
	}

	// Generate a unique ID for the withdrawal (you can replace this with your own logic)
	// Call the UserAppService to create the withdrawal
	err := uc.UserAppService.CreateWithdrawal(c.Request().Context(), request.Amount, request.Wallet, request.JettonName)
	if err != nil {

		log.Printf("[CreateWithdrawal] Error creating withdrawal: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	// Return a success response
	return c.JSON(http.StatusCreated, map[string]string{"message": "Withdrawal created successfully"})
}

// GetWithdrawal handles GET /users/withdrawal/{id}
// @Summary Get a withdrawal by ID
// @Description Retrieve a withdrawal by its unique ID
// @Tags users
// @Produce json
// @Param id path string true "Withdrawal ID"
// @Success 200 {object} Withdrawal "Withdrawal details"
// @Failure 404 {object} map[string]string "Withdrawal not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /users/withdrawal/{id} [get]
func (uc *UserController) GetWithdrawal(ctx echo.Context) error {
	id := ctx.Param("id")

	withdrawal, err := uc.UserAppService.GetWithdrawal(ctx.Request().Context(), id)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, map[string]string{"message": "Withdrawal not found"})
	}

	return ctx.JSON(http.StatusOK, withdrawal)
}

// GetWithdrawalsByWallet handles GET /users/{wallet}/withdrawals
// @Summary Get withdrawals by user wallet
// @Description Retrieve a list of withdrawals for a user by their wallet address
// @Tags users
// @Produce json
// @Param wallet path string true "User Wallet"
// @Param limit query int false "Number of withdrawals to retrieve"
// @Success 200 {array} Withdrawal "List of withdrawals"
// @Failure 500 {object} map[string]string "Error fetching withdrawals"
// @Router /users/{wallet}/withdrawals [get]
func (uc *UserController) GetWithdrawalsByWallet(ctx echo.Context) error {
	wallet := ctx.Param("wallet")

	// Get the limit from query parameters, defaulting to 10 if not provided
	limit := int64(10)
	if l := ctx.QueryParam("limit"); l != "" {
		parsedLimit, err := strconv.ParseInt(l, 10, 64)
		if err == nil {
			limit = parsedLimit
		}
	}

	// Call the UserAppService method with the wallet and limit
	withdrawals, err := uc.UserAppService.GetWithdrawalsByWallet(ctx.Request().Context(), wallet, limit)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"message": "Error fetching withdrawals"})
	}

	return ctx.JSON(http.StatusOK, withdrawals)
}

// GetLast50Withdrawals handles GET /users/withdrawals/last-50
// @Summary Get the last 50 withdrawals
// @Description Retrieve the last 50 withdrawals in the system
// @Tags users
// @Produce json
// @Success 200 {array} Withdrawal "List of the last 50 withdrawals"
// @Failure 500 {object} map[string]string "Error fetching withdrawals"
// @Router /users/withdrawals/last-50 [get]
func (uc *UserController) GetLast50Withdrawals(ctx echo.Context) error {
	withdrawals, err := uc.UserAppService.GetLast50Withdrawals(ctx.Request().Context())
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"message": "Error fetching withdrawals"})
	}

	return ctx.JSON(http.StatusOK, withdrawals)
}

// DeleteWithdrawal handles DELETE /users/withdrawal/{id}
// @Summary Delete a withdrawal by ID
// @Description Delete a withdrawal request by its unique ID
// @Tags users
// @Param id path string true "Withdrawal ID"
// @Success 200 {object} map[string]string "Withdrawal deleted successfully"
// @Failure 404 {object} map[string]string "Withdrawal not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /users/withdrawal/{id} [delete]
func (uc *UserController) DeleteWithdrawal(ctx echo.Context) error {
	id := ctx.Param("id")

	err := uc.UserAppService.WithdrawalService.DeleteWithdrawal(ctx.Request().Context(), id)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, map[string]string{"message": "Withdrawal not found"})
	}

	return ctx.JSON(http.StatusOK, map[string]string{"message": "Withdrawal deleted successfully"})
}

// GetLast50WithdrawalsWithJetton handles GET /users/withdrawals/last-50-with-jetton
// @Summary Get the last 50 withdrawals with jetton
// @Description Retrieve the last 50 withdrawals that include jetton
// @Tags users
// @Produce json
// @Param wallet path string true "User Wallet"
// @Success 200 {array} Withdrawal "List of the last 50 withdrawals with jetton"
// @Failure 500 {object} map[string]string "Error fetching withdrawals with jetton"
// @Router /users/withdrawals/last-50-with-jetton [get]
func (uc *UserController) GetLast50WithdrawalsWithJetton(ctx echo.Context) error {
	// Get the wallet from the path parameter
	wallet := ctx.Param("wallet")

	// Call the service with both context and wallet
	withdrawals, err := uc.UserAppService.GetLast50WithdrawalsWithJetton(ctx.Request().Context(), wallet)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"message": "Error fetching withdrawals with jetton"})
	}

	return ctx.JSON(http.StatusOK, withdrawals)
}

// GetLast50WithdrawalsWithoutJetton handles GET /users/withdrawals/last-50-without-jetton
// @Summary Get the last 50 withdrawals without jetton
// @Description Retrieve the last 50 withdrawals that do not include jetton
// @Tags users
// @Produce json
// @Success 200 {array} Withdrawal "List of the last 50 withdrawals without jetton"
// @Failure 500 {object} map[string]string "Error fetching withdrawals without jetton"
// @Router /users/withdrawals/last-50-without-jetton [get]
func (uc *UserController) GetLast50WithdrawalsWithoutJetton(ctx echo.Context) error {
	withdrawals, err := uc.UserAppService.GetLast50WithdrawalsWithoutJetton(ctx.Request().Context())
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"message": "Error fetching withdrawals without jetton"})
	}

	return ctx.JSON(http.StatusOK, withdrawals)
}
