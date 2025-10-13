package handlers

import (
	"net/http"

	"go.uber.org/zap"
)

type AdminHandler struct {
	logger *zap.Logger
}

func NewAdminHandler(logger *zap.Logger) *AdminHandler {
	return &AdminHandler{
		logger: logger,
	}
}

func (h *AdminHandler) ShowAdminDashboard(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/admin.html")
}
