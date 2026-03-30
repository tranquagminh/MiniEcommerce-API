package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"product-service/internal/infrastructure/postgres"
)

type QAHandler struct {
	repo *postgres.QARepository
}

func NewQAHandler(repo *postgres.QARepository) *QAHandler {
	return &QAHandler{repo: repo}
}

// QAResponse is the DTO for Q&A responses with flattened fields
type QAResponse struct {
	ID           uint    `json:"id"`
	UserID       uint    `json:"user_id"`
	ProductID    uint    `json:"product_id"`
	ProductName  string  `json:"product_name,omitempty"`
	Question     string  `json:"question"`
	Answer       string  `json:"answer,omitempty"`
	Status       string  `json:"status"`
	AnsweredBy   *uint   `json:"answered_by,omitempty"`
	AnsweredAt   *string `json:"answered_at,omitempty"`
	HelpfulCount int     `json:"helpful_count"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

func toQAResponse(qa *postgres.ProductQAModel) QAResponse {
	resp := QAResponse{
		ID:           qa.ID,
		UserID:       qa.UserID,
		ProductID:    qa.ProductID,
		Question:     qa.Question,
		Answer:       qa.Answer,
		Status:       qa.Status,
		AnsweredBy:   qa.AnsweredBy,
		HelpfulCount: qa.HelpfulCount,
		CreatedAt:    qa.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    qa.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if qa.AnsweredAt != nil {
		formatted := qa.AnsweredAt.Format("2006-01-02T15:04:05Z07:00")
		resp.AnsweredAt = &formatted
	}
	if qa.Product != nil {
		resp.ProductName = qa.Product.Name
	}
	return resp
}

func toQAResponses(qas []postgres.ProductQAModel) []QAResponse {
	responses := make([]QAResponse, len(qas))
	for i, qa := range qas {
		responses[i] = toQAResponse(&qa)
	}
	return responses
}

// CreateQuestionRequest represents the request body for creating a question
type CreateQuestionRequest struct {
	UserID    uint   `json:"user_id"`
	ProductID uint   `json:"product_id"`
	Question  string `json:"question"`
}

// CreateQuestion handles POST /qa
func (h *QAHandler) CreateQuestion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if req.Question == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Question cannot be empty"})
		return
	}

	qa := &postgres.ProductQAModel{
		UserID:    req.UserID,
		ProductID: req.ProductID,
		Question:  req.Question,
		Status:    postgres.QAStatusPending,
	}

	if err := h.repo.Create(r.Context(), qa); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create question"})
		return
	}

	respondJSON(w, http.StatusCreated, qa)
}

// GetQA handles GET /qa/{id}
func (h *QAHandler) GetQA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/qa/")
	id, err := strconv.ParseUint(path, 10, 32)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid Q&A ID"})
		return
	}

	qa, err := h.repo.GetByID(r.Context(), uint(id))
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Q&A not found"})
		return
	}

	respondJSON(w, http.StatusOK, qa)
}

// ListQA handles GET /qa (admin)
func (h *QAHandler) ListQA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	limit, _ := strconv.Atoi(query.Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	filter := postgres.QAFilter{
		Status: query.Get("status"),
	}

	if productID := query.Get("product_id"); productID != "" {
		if id, err := strconv.ParseUint(productID, 10, 32); err == nil {
			pid := uint(id)
			filter.ProductID = &pid
		}
	}

	qas, total, err := h.repo.List(r.Context(), filter, page, limit)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to list Q&A"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":        toQAResponses(qas),
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": (total + int64(limit) - 1) / int64(limit),
	})
}

// GetProductQA handles GET /products/{id}/qa (public - answered only)
func (h *QAHandler) GetProductQA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/products/")
	path = strings.TrimSuffix(path, "/qa")
	productID, err := strconv.ParseUint(path, 10, 32)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid product ID"})
		return
	}

	query := r.URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	limit, _ := strconv.Atoi(query.Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Only show answered questions for public
	includeUnanswered := query.Get("include_pending") == "true"

	qas, total, err := h.repo.GetByProductID(r.Context(), uint(productID), includeUnanswered, page, limit)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to get Q&A"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":        qas,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": (total + int64(limit) - 1) / int64(limit),
	})
}

// AnswerQuestion handles POST /qa/{id}/answer
func (h *QAHandler) AnswerQuestion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/qa/")
	path = strings.TrimSuffix(path, "/answer")
	id, err := strconv.ParseUint(path, 10, 32)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid Q&A ID"})
		return
	}

	var req struct {
		Answer     string `json:"answer"`
		AnsweredBy uint   `json:"answered_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if req.Answer == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Answer cannot be empty"})
		return
	}

	if err := h.repo.Answer(r.Context(), uint(id), req.Answer, req.AnsweredBy); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to answer question"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Question answered successfully"})
}

// DeleteQA handles DELETE /qa/{id}
func (h *QAHandler) DeleteQA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/qa/")
	id, err := strconv.ParseUint(path, 10, 32)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid Q&A ID"})
		return
	}

	if err := h.repo.Delete(r.Context(), uint(id)); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete Q&A"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Q&A deleted successfully"})
}

// GetPendingCount handles GET /qa/pending-count
func (h *QAHandler) GetPendingCount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	count, err := h.repo.GetPendingCount(r.Context())
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to get pending count"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]int64{"pending_count": count})
}
