package main

import (
	"log"
	"net/http"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"onepagediary/internal/config"
	"onepagediary/internal/db"
	"onepagediary/internal/handlers"
	"onepagediary/internal/models"
)

func main() {
	cfg := config.Load()

	dbConn, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("db open failed: %v", err)
	}

	if err := ensureAdminUser(dbConn, cfg.AdminUser, cfg.AdminPassword); err != nil {
		log.Fatalf("seed admin user failed: %v", err)
	}

	server := &handlers.Server{DB: dbConn, Config: cfg}
	log.Printf("server listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, server.Routes()); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func ensureAdminUser(dbConn *gorm.DB, username, password string) error {
	var count int64
	if err := dbConn.Model(&models.User{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return dbConn.Create(&models.User{Username: username, PasswordHash: string(hash)}).Error
}
