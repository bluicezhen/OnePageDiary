package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"onepagediary/internal/config"
	"onepagediary/internal/db"
	"onepagediary/internal/models"
)

type entryResponsePayload struct {
	Date     string `json:"date"`
	Content  string `json:"content"`
	Revision uint64 `json:"revision"`
	Deleted  bool   `json:"deleted"`
}

type conflictResponse struct {
	Error string               `json:"error"`
	Entry entryResponsePayload `json:"entry"`
}

type syncChangesResponse struct {
	Events     []struct {
		ID       uint64 `json:"id"`
		Date     string `json:"date"`
		Action   string `json:"action"`
		Revision uint64 `json:"revision"`
	} `json:"events"`
	LastCursor uint64 `json:"lastCursor"`
}

func setupTestServer(t *testing.T) (*httptest.Server, config.Config, *gorm.DB, models.User) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "diary.db")

	dbConn, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	user := models.User{
		Username: "tester",
	}
	password := "pass123"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user.PasswordHash = string(hash)
	if err := dbConn.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	cfg := config.Config{
		Addr:            ":0",
		DBPath:          dbPath,
		JWTSecret:       "test-secret",
		AdminUser:       "",
		AdminPassword:   "",
		CORSOrigins:     "*",
		ServerVersion:   "test",
		ProtocolVersion: "v1",
	}

	srv := &Server{DB: dbConn, Config: cfg}
	httpServer := httptest.NewServer(srv.Routes())

	return httpServer, cfg, dbConn, user
}

func authToken(t *testing.T, secret string, user models.User) string {
	t.Helper()
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func TestLoginSuccess(t *testing.T) {
	srv, cfg, _, _ := setupTestServer(t)
	defer srv.Close()

	payload := map[string]string{"username": "tester", "password": "pass123"}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(srv.URL+"/api/v1/auth/login", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("login request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var data map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if data["token"] == "" {
		t.Fatalf("expected token, got empty")
	}

	_, err = jwt.Parse(data["token"], func(token *jwt.Token) (any, error) {
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil {
		t.Fatalf("token parse failed: %v", err)
	}
}

func TestEntryFlowAndConflict(t *testing.T) {
	srv, cfg, _, user := setupTestServer(t)
	defer srv.Close()

	token := authToken(t, cfg.JWTSecret, user)
	client := &http.Client{}

	date := "2024-01-01"

	unauthResp, err := client.Get(srv.URL + "/api/v1/entries/" + date)
	if err != nil {
		t.Fatalf("unauth get: %v", err)
	}
	if unauthResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", unauthResp.StatusCode)
	}

	createPayload := map[string]any{"content": "hello", "baseRevision": 0}
	createBody, _ := json.Marshal(createPayload)
	createReq, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/entries/"+date, bytes.NewBuffer(createBody))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", "application/json")

	createResp, err := client.Do(createReq)
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", createResp.StatusCode)
	}

	var created entryResponsePayload
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.Revision != 1 {
		t.Fatalf("expected revision 1, got %d", created.Revision)
	}

	conflictPayload := map[string]any{"content": "server", "baseRevision": 0}
	conflictBody, _ := json.Marshal(conflictPayload)
	conflictReq, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/entries/"+date, bytes.NewBuffer(conflictBody))
	conflictReq.Header.Set("Authorization", "Bearer "+token)
	conflictReq.Header.Set("Content-Type", "application/json")

	conflictResp, err := client.Do(conflictReq)
	if err != nil {
		t.Fatalf("conflict request: %v", err)
	}
	defer conflictResp.Body.Close()

	if conflictResp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", conflictResp.StatusCode)
	}

	var conflict conflictResponse
	if err := json.NewDecoder(conflictResp.Body).Decode(&conflict); err != nil {
		t.Fatalf("decode conflict: %v", err)
	}
	if conflict.Entry.Revision != 1 {
		t.Fatalf("expected conflict revision 1, got %d", conflict.Entry.Revision)
	}

	updatePayload := map[string]any{"content": "updated", "baseRevision": 1}
	updateBody, _ := json.Marshal(updatePayload)
	updateReq, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/entries/"+date, bytes.NewBuffer(updateBody))
	updateReq.Header.Set("Authorization", "Bearer "+token)
	updateReq.Header.Set("Content-Type", "application/json")

	updateResp, err := client.Do(updateReq)
	if err != nil {
		t.Fatalf("update request: %v", err)
	}
	defer updateResp.Body.Close()

	if updateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", updateResp.StatusCode)
	}

	var updated entryResponsePayload
	if err := json.NewDecoder(updateResp.Body).Decode(&updated); err != nil {
		t.Fatalf("decode update: %v", err)
	}
	if updated.Revision != 2 {
		t.Fatalf("expected revision 2, got %d", updated.Revision)
	}

	deleteReq, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/entries/"+date+"?baseRevision=2", nil)
	deleteReq.Header.Set("Authorization", "Bearer "+token)
	deleteResp, err := client.Do(deleteReq)
	if err != nil {
		t.Fatalf("delete request: %v", err)
	}
	defer deleteResp.Body.Close()

	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", deleteResp.StatusCode)
	}

	var deleted entryResponsePayload
	if err := json.NewDecoder(deleteResp.Body).Decode(&deleted); err != nil {
		t.Fatalf("decode delete: %v", err)
	}
	if !deleted.Deleted || deleted.Revision != 3 {
		t.Fatalf("expected deleted revision 3, got deleted=%v revision=%d", deleted.Deleted, deleted.Revision)
	}
}

func TestSyncChanges(t *testing.T) {
	srv, cfg, _, user := setupTestServer(t)
	defer srv.Close()

	token := authToken(t, cfg.JWTSecret, user)
	client := &http.Client{}
	date := "2024-01-02"

	createPayload := map[string]any{"content": "hello", "baseRevision": 0}
	createBody, _ := json.Marshal(createPayload)
	createReq, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/entries/"+date, bytes.NewBuffer(createBody))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", "application/json")

	createResp, err := client.Do(createReq)
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}
	createResp.Body.Close()

	updatePayload := map[string]any{"content": "hello2", "baseRevision": 1}
	updateBody, _ := json.Marshal(updatePayload)
	updateReq, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/entries/"+date, bytes.NewBuffer(updateBody))
	updateReq.Header.Set("Authorization", "Bearer "+token)
	updateReq.Header.Set("Content-Type", "application/json")

	updateResp, err := client.Do(updateReq)
	if err != nil {
		t.Fatalf("update entry: %v", err)
	}
	updateResp.Body.Close()

	syncReq, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/sync/changes?after=0", nil)
	syncReq.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(syncReq)
	if err != nil {
		t.Fatalf("sync request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var changes syncChangesResponse
	if err := json.NewDecoder(resp.Body).Decode(&changes); err != nil {
		t.Fatalf("decode sync: %v", err)
	}

	if len(changes.Events) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(changes.Events))
	}
	if changes.LastCursor == 0 {
		t.Fatalf("expected lastCursor > 0")
	}
}
