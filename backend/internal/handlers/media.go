package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"pixellife-tracker/internal/models"
	"pixellife-tracker/internal/services"
)

type MediaHandler struct {
	svc *services.MediaService
}

func NewMediaHandler(svc *services.MediaService) *MediaHandler {
	return &MediaHandler{svc: svc}
}

func (h *MediaHandler) RegisterRoutes(r fiber.Router) {
	r.Post("/", h.Create)
	r.Get("/", h.GetAll)
	r.Get("/recent", h.GetRecent)
	r.Put("/:id", h.Update)
	r.Delete("/:id", h.Delete)
}

func (h *MediaHandler) Create(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	var req models.MediaItem
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := h.svc.Create(userID, &req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(req)
}

func (h *MediaHandler) GetAll(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	mediaType := c.Query("type")
	items, err := h.svc.GetByUser(userID, mediaType)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(items)
}

func (h *MediaHandler) GetRecent(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	items, err := h.svc.GetRecent(userID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(items)
}

func (h *MediaHandler) Update(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	var req models.MediaItem
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	req.ID = id
	req.UserID = userID
	if err := h.svc.Update(&req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(req)
}

func (h *MediaHandler) Delete(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	if err := h.svc.Delete(userID, id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}