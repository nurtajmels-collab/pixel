package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"pixellife-tracker/internal/models"
	"pixellife-tracker/internal/services"
)

type StudyHandler struct {
	svc *services.StudyService
}

func NewStudyHandler(svc *services.StudyService) *StudyHandler {
	return &StudyHandler{svc: svc}
}

func (h *StudyHandler) RegisterRoutes(r fiber.Router) {
	r.Post("/sessions", h.CreateSession)
	r.Get("/sessions", h.GetSessions)
	r.Get("/today", h.GetTodayMinutes)
	r.Post("/scores", h.AddScore)
	r.Get("/scores/:category", h.GetScores)
	r.Delete("/scores/:id", h.DeleteScore)
	r.Post("/assessments", h.AddAssessment)
	r.Get("/assessments", h.GetAssessments)
	r.Get("/summary", h.GetSummary)
	r.Put("/weights", h.SetWeights)
}

func (h *StudyHandler) AddAssessment(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	var assessment models.StudyAssessment
	if err := c.BodyParser(&assessment); err != nil { return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()}) }
	if assessment.Subject == "" || (assessment.AssessmentType != "FA" && assessment.AssessmentType != "SAU" && assessment.AssessmentType != "SAT") || assessment.MaxScore <= 0 || assessment.Score < 0 || assessment.Score > assessment.MaxScore {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "subject, assessment type, score and max_score are required"})
	}
	if err := h.svc.AddAssessment(userID, &assessment); err != nil { return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()}) }
	return c.Status(fiber.StatusCreated).JSON(assessment)
}

func (h *StudyHandler) GetAssessments(c *fiber.Ctx) error {
	assessments, err := h.svc.GetAssessments(c.Locals("user_id").(int64), c.Query("subject"))
	if err != nil { return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()}) }
	return c.JSON(assessments)
}

func (h *StudyHandler) GetSummary(c *fiber.Ctx) error {
	subject := c.Query("subject")
	if subject == "" { return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "subject is required"}) }
	summary, err := h.svc.GetSummary(c.Locals("user_id").(int64), subject)
	if err != nil { return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()}) }
	return c.JSON(summary)
}

func (h *StudyHandler) SetWeights(c *fiber.Ctx) error {
	var req struct { Subject string `json:"subject"`; FA float64 `json:"fa"`; SAU float64 `json:"sau"`; SAT float64 `json:"sat"` }
	if err := c.BodyParser(&req); err != nil || req.Subject == "" || req.FA < 0 || req.SAU < 0 || req.SAT < 0 || req.FA+req.SAU+req.SAT < 0.999 || req.FA+req.SAU+req.SAT > 1.001 { return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "weights must be non-negative and sum to 1"}) }
	if err := h.svc.SetWeights(c.Locals("user_id").(int64), req.Subject, req.FA, req.SAU, req.SAT); err != nil { return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()}) }
	return c.JSON(req)
}

func (h *StudyHandler) CreateSession(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	var req struct {
		models.StudySession
		Source string `json:"source"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if req.Source != "pomodoro" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "study sessions are created only when a Pomodoro finishes"})
	}
	if err := h.svc.CreateSession(userID, &req.StudySession); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(req.StudySession)
}

func (h *StudyHandler) GetSessions(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	sessions, err := h.svc.GetSessions(userID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(sessions)
}

func (h *StudyHandler) GetTodayMinutes(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	minutes, err := h.svc.GetTodayMinutes(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"minutes": minutes})
}

func (h *StudyHandler) AddScore(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	var req struct {
		Category  string  `json:"category"`
		Subject   string  `json:"subject"`
		Component string  `json:"component"`
		Score     float64 `json:"score"`
		MaxScore  float64 `json:"max_score"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if req.Score < 0 || req.MaxScore <= 0 || req.Score > req.MaxScore {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "score must be between zero and max_score"})
	}
	if err := h.svc.AddScore(userID, req.Category, req.Subject, req.Component, req.Score, req.MaxScore); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusCreated)
}

func (h *StudyHandler) GetScores(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	category := c.Params("category")
	scores, err := h.svc.GetScores(userID, category)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(scores)
}

func (h *StudyHandler) DeleteScore(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int64)
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid score id"})
	}
	if err := h.svc.DeleteScore(userID, id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}