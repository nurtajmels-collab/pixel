package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"pixellife-tracker/internal/models"
	"pixellife-tracker/internal/services"
)

type HobbyHandler struct {
	svc *services.HobbyService
}

func NewHobbyHandler(svc *services.HobbyService) *HobbyHandler {
	return &HobbyHandler{svc: svc}
}

func (h *HobbyHandler) RegisterRoutes(r fiber.Router) {
	r.Post("/streaks", h.CreateStreak)
	r.Get("/streaks", h.GetStreaks)
	r.Post("/streaks/:id/complete", h.CompleteStreak)
	r.Post("/sessions", h.AddSession)
	r.Get("/sessions", h.GetSessions)
}

func (h *HobbyHandler) CreateStreak(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	var req models.HobbyStreak
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := h.svc.CreateStreak(userID, &req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(req)
}

func (h *HobbyHandler) GetStreaks(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	streaks, err := h.svc.GetStreaks(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(streaks)
}

func (h *HobbyHandler) CompleteStreak(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	if err := h.svc.CompleteStreak(userID, id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusOK)
}

func (h *HobbyHandler) AddSession(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	var req models.HobbySession
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := h.svc.AddSession(userID, &req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(req)
}

func (h *HobbyHandler) GetSessions(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	sessions, err := h.svc.GetSessions(userID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(sessions)
}

type F1Handler struct {
	svc *services.F1Service
}

func NewF1Handler(svc *services.F1Service) *F1Handler {
	return &F1Handler{svc: svc}
}

func (h *F1Handler) RegisterRoutes(r fiber.Router) {
	r.Post("/events", h.CreateEvent)
	r.Get("/events", h.GetAll)
	r.Get("/events/upcoming", h.GetUpcoming)
	r.Delete("/events/:id", h.Delete)
}

func (h *F1Handler) CreateEvent(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	var req models.F1Event
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := h.svc.CreateEvent(userID, &req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(req)
}

func (h *F1Handler) GetAll(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	events, err := h.svc.GetAll(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(events)
}

func (h *F1Handler) GetUpcoming(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	limit, _ := strconv.Atoi(c.Query("limit", "5"))
	events, err := h.svc.GetUpcoming(userID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(events)
}

func (h *F1Handler) Delete(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	if err := h.svc.Delete(userID, id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}