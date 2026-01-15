package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"onepagediary/internal/config"
	"onepagediary/internal/models"
)

type Server struct {
	DB     *gorm.DB
	Config config.Config
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/health", s.handleHealth)
	mux.HandleFunc("/api/v1/version", s.handleVersion)
	mux.HandleFunc("/api/v1/auth/login", s.handleLogin)

	protected := http.NewServeMux()
	protected.HandleFunc("/api/v1/sync/changes", s.handleSyncChanges)
	protected.HandleFunc("/api/v1/entries/", s.handleEntry)

	mux.Handle("/api/v1/sync/changes", WithAuth(s.Config.JWTSecret, protected))
	mux.Handle("/api/v1/entries/", WithAuth(s.Config.JWTSecret, protected))

	return WithCORS(mux, s.Config.CORSOrigins)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"time":    time.Now().Format(time.RFC3339),
		"version": s.Config.ServerVersion,
	})
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"serverVersion":   s.Config.ServerVersion,
		"protocolVersion": s.Config.ProtocolVersion,
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
		return
	}

	var user models.User
	if err := s.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "invalid_credentials"})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "invalid_credentials"})
		return
	}

	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.Config.JWTSecret))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "token_sign_failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"token": signed})
}

func (s *Server) handleEntry(w http.ResponseWriter, r *http.Request) {
	date := strings.TrimPrefix(r.URL.Path, "/api/v1/entries/")
	if date == "" || strings.Contains(date, "/") {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_date"})
		return
	}

	if _, err := time.Parse("2006-01-02", date); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_date"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getEntry(w, r, date)
	case http.MethodPut:
		s.putEntry(w, r, date)
	case http.MethodDelete:
		s.deleteEntry(w, r, date)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) getEntry(w http.ResponseWriter, r *http.Request, date string) {
	var entry models.Entry
	if err := s.DB.Where("date = ?", date).First(&entry).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "not_found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "db_error"})
		return
	}

	if entry.Deleted {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error":   "deleted",
			"date":    entry.Date,
			"revision": entry.Revision,
		})
		return
	}

	writeJSON(w, http.StatusOK, entryResponse(entry))
}

func (s *Server) putEntry(w http.ResponseWriter, r *http.Request, date string) {
	force := r.URL.Query().Get("force") == "1"
	var req struct {
		Content      string `json:"content"`
		BaseRevision uint64 `json:"baseRevision"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
		return
	}

	entry, conflict, err := s.upsertEntry(date, req.Content, req.BaseRevision, force)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "db_error"})
		return
	}

	if conflict != nil {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error": "conflict",
			"entry": entryResponse(*conflict),
		})
		return
	}

	writeJSON(w, http.StatusOK, entryResponse(entry))
}

func (s *Server) deleteEntry(w http.ResponseWriter, r *http.Request, date string) {
	force := r.URL.Query().Get("force") == "1"
	baseRevision := parseUint(r.URL.Query().Get("baseRevision"))

	entry, conflict, err := s.removeEntry(date, baseRevision, force)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "not_found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "db_error"})
		return
	}

	if conflict != nil {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error": "conflict",
			"entry": entryResponse(*conflict),
		})
		return
	}

	writeJSON(w, http.StatusOK, entryResponse(entry))
}

func (s *Server) handleSyncChanges(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	after := parseUint(r.URL.Query().Get("after"))
	limit := int(parseUint(r.URL.Query().Get("limit")))
	if limit <= 0 || limit > 1000 {
		limit = 500
	}

	var events []models.SyncEvent
	if err := s.DB.Where("id > ?", after).Order("id asc").Limit(limit).Find(&events).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "db_error"})
		return
	}

	lastCursor := after
	if len(events) > 0 {
		lastCursor = events[len(events)-1].ID
	}

	payload := make([]map[string]any, 0, len(events))
	for _, evt := range events {
		payload = append(payload, map[string]any{
			"id":        evt.ID,
			"date":      evt.Date,
			"action":    evt.Action,
			"revision":  evt.Revision,
			"createdAt": evt.CreatedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"events":     payload,
		"lastCursor": lastCursor,
	})
}

func (s *Server) upsertEntry(date string, content string, baseRevision uint64, force bool) (models.Entry, *models.Entry, error) {
	var entry models.Entry

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("date = ?", date).First(&entry)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			if !force && baseRevision != 0 {
				conflict := models.Entry{Date: date, Revision: 0, Deleted: false}
				entry = conflict
				return fmt.Errorf("conflict")
			}

			entry = models.Entry{
				Date:     date,
				Content:  content,
				Revision: 1,
				Deleted:  false,
			}
			if err := tx.Create(&entry).Error; err != nil {
				return err
			}
			return tx.Create(&models.SyncEvent{Date: date, Action: "upsert", Revision: entry.Revision}).Error
		}

		if result.Error != nil {
			return result.Error
		}

		if !force && baseRevision != entry.Revision {
			return fmt.Errorf("conflict")
		}

		entry.Content = content
		entry.Deleted = false
		entry.Revision++
		entry.UpdatedAt = time.Now()

		if err := tx.Save(&entry).Error; err != nil {
			return err
		}

		return tx.Create(&models.SyncEvent{Date: date, Action: "upsert", Revision: entry.Revision}).Error
	})

	if err != nil {
		if err.Error() == "conflict" {
			conflictEntry := entry
			return models.Entry{}, &conflictEntry, nil
		}
		return models.Entry{}, nil, err
	}

	return entry, nil, nil
}

func (s *Server) removeEntry(date string, baseRevision uint64, force bool) (models.Entry, *models.Entry, error) {
	var entry models.Entry

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("date = ?", date).First(&entry)
		if result.Error != nil {
			return result.Error
		}

		if !force && baseRevision != entry.Revision {
			return fmt.Errorf("conflict")
		}

		if entry.Deleted {
			return nil
		}

		entry.Deleted = true
		entry.Revision++
		entry.UpdatedAt = time.Now()
		if err := tx.Save(&entry).Error; err != nil {
			return err
		}

		return tx.Create(&models.SyncEvent{Date: date, Action: "delete", Revision: entry.Revision}).Error
	})

	if err != nil {
		if err.Error() == "conflict" {
			conflictEntry := entry
			return models.Entry{}, &conflictEntry, nil
		}
		return models.Entry{}, nil, err
	}

	return entry, nil, nil
}

func entryResponse(entry models.Entry) map[string]any {
	return map[string]any{
		"date":      entry.Date,
		"content":   entry.Content,
		"revision":  entry.Revision,
		"deleted":   entry.Deleted,
		"updatedAt": entry.UpdatedAt.Format(time.RFC3339),
		"createdAt": entry.CreatedAt.Format(time.RFC3339),
	}
}

func parseUint(raw string) uint64 {
	if raw == "" {
		return 0
	}
	val, _ := strconv.ParseUint(raw, 10, 64)
	return val
}

func readJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
