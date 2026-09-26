package handlers

import (
	"github.com/gofiber/fiber/v2"
	"pixellife-tracker/internal/services"
)

type DashboardHandler struct {
	svc *services.RadarService
}

func NewDashboardHandler(svc *services.RadarService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

func (h *DashboardHandler) RegisterRoutes(r fiber.Router) {
	r.Get("/", h.GetDashboard)
	r.Post("/recalculate", h.Recalculate)
}

func (h *DashboardHandler) GetDashboard(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	data, err := h.svc.GetDashboardData(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(data)
}

func (h *DashboardHandler) Recalculate(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	scores, err := h.svc.CalculateAndStore(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(scores)
}