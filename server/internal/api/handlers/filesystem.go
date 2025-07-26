package handlers

import (
	"encoding/json"
	"net/http"

	"gitweb/server/internal/filesystem"
)

type FilesystemHandler struct {
	fsService *filesystem.Service
}

func NewFilesystemHandler(restrictToUserHome bool) *FilesystemHandler {
	return &FilesystemHandler{
		fsService: filesystem.NewService(restrictToUserHome),
	}
}

func (h *FilesystemHandler) BrowseDirectory(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")

	listing, err := h.fsService.BrowseDirectory(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(listing)
}

func (h *FilesystemHandler) GetVolumeRoots(w http.ResponseWriter, r *http.Request) {
	roots, err := h.fsService.GetVolumeRoots()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"roots": roots,
	})
}