package config

import "os"

type Config struct {
	Addr           string
	DBPath         string
	JWTSecret      string
	AdminUser      string
	AdminPassword  string
	CORSOrigins    string
	ServerVersion  string
	ProtocolVersion string
}

func Load() Config {
	cfg := Config{
		Addr:           getEnv("APP_ADDR", ":8080"),
		DBPath:         getEnv("APP_DB_PATH", "./data/diary.db"),
		JWTSecret:      getEnv("APP_JWT_SECRET", "dev-secret"),
		AdminUser:      getEnv("APP_ADMIN_USER", "admin"),
		AdminPassword:  getEnv("APP_ADMIN_PASS", "admin"),
		CORSOrigins:    getEnv("APP_CORS_ORIGINS", "*"),
		ServerVersion:  getEnv("APP_SERVER_VERSION", "0.1.0"),
		ProtocolVersion: getEnv("APP_PROTOCOL_VERSION", "v1"),
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
