package controllers

import (
	"net/http"

	"github.com/Peranum/tg-dice/internal/promocodes/domain/services"
	"github.com/Peranum/tg-dice/internal/promocodes/infrastructure/entity"
	"github.com/labstack/echo/v4"
)

// PromoCodeController handles HTTP requests for promocodes
type PromoCodeController struct {
	service *services.PromoCodeService
}

// NewPromoCodeController creates a new PromoCodeController
func NewPromoCodeController(service *services.PromoCodeService) *PromoCodeController {
	return &PromoCodeController{service: service}
}

// CreatePromoCode creates a new promocode
// @Summary Create a new promocode
// @Description Create a promocode with details like code, type, amount, and activations
// @Tags PromoCodes
// @Accept json
// @Produce json
// @Param request body entity.PromoCodeEntity true "Promocode details"
// @Success 201 {string} string "Promocode created successfully"
// @Failure 400 {string} string "Invalid request payload"
// @Failure 500 {string} string "Internal server error"
// @Router /promocodes/create [post]
func (c *PromoCodeController) CreatePromoCode(ctx echo.Context) error {
	var promo entity.PromoCodeEntity
	if err := ctx.Bind(&promo); err != nil {
		return ctx.JSON(http.StatusBadRequest, "Invalid request payload")
	}

	err := c.service.CreatePromoCode(ctx.Request().Context(), &promo)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, err.Error())
	}

	return ctx.JSON(http.StatusCreated, "Promocode created successfully")
}

// ActivatePromoCode activates a promocode
// @Summary Activate a promocode
// @Description Activate a promocode by providing the code and user wallet
// @Tags PromoCodes
// @Accept json
// @Produce json
// @Param request body map[string]string true "Activation request (wallet and code)"
// @Success 200 {string} string "Promocode activated successfully"
// @Failure 400 {string} string "Invalid request payload"
// @Failure 404 {string} string "Promocode or user not found"
// @Failure 500 {string} string "Internal server error"
// @Router /promocodes/activate [post]
func (c *PromoCodeController) ActivatePromoCode(ctx echo.Context) error {
	var request struct {
		Wallet string `json:"wallet"`
		Code   string `json:"code"`
	}

	if err := ctx.Bind(&request); err != nil {
		return ctx.JSON(http.StatusBadRequest, "Invalid request payload")
	}

	err := c.service.ActivatePromoCode(ctx.Request().Context(), request.Wallet, request.Code)
	if err != nil {
		if err.Error() == "user not found" || err.Error() == "promocode not found or activations exhausted" {
			return ctx.JSON(http.StatusNotFound, err.Error())
		}
		return ctx.JSON(http.StatusInternalServerError, err.Error())
	}

	return ctx.JSON(http.StatusOK, "Promocode activated successfully")
}

// ListActivePromoCodes lists all active promocodes
// @Summary List active promocodes
// @Description Retrieve all active promocodes
// @Tags PromoCodes
// @Produce json
// @Success 200 {array} entity.PromoCodeEntity "List of active promocodes"
// @Failure 500 {string} string "Internal server error"
// @Router /promocodes/active [get]
func (c *PromoCodeController) ListActivePromoCodes(ctx echo.Context) error {
	promocodes, err := c.service.ListActivePromoCodes(ctx.Request().Context())
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, err.Error())
	}

	return ctx.JSON(http.StatusOK, promocodes)
}

// GetPromoCode retrieves a promocode by its code
// @Summary Get promocode details
// @Description Retrieve details of a specific promocode by its code
// @Tags PromoCodes
// @Produce json
// @Param code path string true "Promocode code"
// @Success 200 {object} entity.PromoCodeEntity "Promocode details"
// @Failure 404 {string} string "Promocode not found"
// @Failure 500 {string} string "Internal server error"
// @Router /promocodes/{code} [get]
func (c *PromoCodeController) GetPromoCode(ctx echo.Context) error {
	code := ctx.Param("code")
	promo, err := c.service.GetPromoCodeByCode(ctx.Request().Context(), code)
	if err != nil {
		if err.Error() == "promocode not found" {
			return ctx.JSON(http.StatusNotFound, err.Error())
		}
		return ctx.JSON(http.StatusInternalServerError, err.Error())
	}

	return ctx.JSON(http.StatusOK, promo)
}

// ExpirePromoCodes expires all outdated promocodes
// @Summary Expire outdated promocodes
// @Description Mark all expired promocodes as expired
// @Tags PromoCodes
// @Produce json
// @Success 200 {string} string "Expired promocodes successfully"
// @Failure 500 {string} string "Internal server error"
// @Router /promocodes/expire [post]
func (c *PromoCodeController) ExpirePromoCodes(ctx echo.Context) error {
	err := c.service.ExpirePromocodes(ctx.Request().Context())
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, err.Error())
	}

	return ctx.JSON(http.StatusOK, "Expired promocodes successfully")
}
