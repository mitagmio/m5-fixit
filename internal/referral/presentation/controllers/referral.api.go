package controllers

import (
	"net/http"
	"strconv"

	"github.com/Peranum/tg-dice/internal/referral/domain/services"
	"github.com/labstack/echo/v4"
)

type ReferralController struct {
	ReferralService *services.ReferralService
}

// NewReferralController создает новый контроллер реферальной системы
func NewReferralController(referralService *services.ReferralService) *ReferralController {
	return &ReferralController{
		ReferralService: referralService,
	}
}

// GetReferralsByLevelsHandler обрабатывает запрос на получение рефералов по уровням
// @Summary Получение рефералов по уровням
// @Description Возвращает список рефералов, разделенных по уровням (level1, level2, level3)
// @Tags referrals
// @Accept json
// @Produce json
// @Param wallet query string true "Wallet address"
// @Success 200 {object} map[string][]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /referrals/levels [get]
func (rc *ReferralController) GetReferralsByLevelsHandler(c echo.Context) error {
	wallet := c.QueryParam("wallet")
	if wallet == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Wallet is required"})
	}

	levels, err := rc.ReferralService.GetReferralsByWallet(c.Request().Context(), wallet)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, levels)
}

// GetReferralsByLevelHandler обрабатывает запрос на получение рефералов по уровню
// @Summary Получение рефералов по уровню
// @Description Возвращает список рефералов на заданном уровне
// @Tags referrals
// @Accept json
// @Produce json
// @Param referral_code query string true "Referral code"
// @Param level query int true "Level (1, 2, or 3)"
// @Success 200 {array} odm_entities.UserEntity
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /referrals/level [get]
func (rc *ReferralController) GetReferralsByLevelHandler(c echo.Context) error {
	referralCode := c.QueryParam("referral_code")
	if referralCode == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Referral code is required"})
	}

	levelParam := c.QueryParam("level")
	if levelParam == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Level is required"})
	}

	level, err := strconv.Atoi(levelParam)
	if err != nil || level <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid level"})
	}

	referrals, err := rc.ReferralService.GetReferralsByLevel(c.Request().Context(), referralCode, level)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, referrals)
}

// GetTotalReferralsHandler обрабатывает запрос на получение общего количества рефералов
// @Summary Получение общего количества рефералов
// @Description Возвращает общее количество рефералов для данного реферального кода
// @Tags referrals
// @Accept json
// @Produce json
// @Param referral_code query string true "Referral code"
// @Success 200 {object} map[string]int
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /referrals/total [get]
func (rc *ReferralController) GetTotalReferralsHandler(c echo.Context) error {
	referralCode := c.QueryParam("referral_code")
	if referralCode == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Referral code is required"})
	}

	total, err := rc.ReferralService.GetTotalReferrals(c.Request().Context(), referralCode)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]int{"total_referrals": total})
}
