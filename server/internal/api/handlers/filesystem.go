package handlers

import (
	"encoding/json"
	"net/http"

	"gitweb/server/internal/filesystem"

	"github.com/go-chi/chi/v5"
)

type FilesystemHandler struct {
	fsService *filesystem.Service
}

func NewFilesystemHandler(restrictToUserHome bool) *FilesystemHandler {
	return &FilesystemHandler{
		fsService: filesystem.NewService(restrictToUserHome),
	}
}

// @Summary      Browse directory
// @Description  Browse a directory on the filesystem
// @Tags         filesystem
// @Produce      json
// @Param        path   query   string  false  "Directory path to browse"
// @Success      200    {object} models.DirectoryListing
// @Failure      400    {string} string "Bad request"
// @Security     BearerAuth
// @Router       /api/filesystem/browse [get]
func (h *FilesystemHandler) BrowseDirectory(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = chi.URLParam(r, "*")
	}

	listing, err := h.fsService.BrowseDirectory(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(listing)
}

// @Summary      List volume roots
// @Description  List allowed root paths/volumes
// @Tags         filesystem
// @Produce      json
// @Success      200    {object} map[string][]string
// @Security     BearerAuth
// @Router       /api/filesystem/roots [get]
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
