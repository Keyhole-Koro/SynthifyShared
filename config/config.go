package config

import (
	"fmt"
	"os"

	"github.com/synthify/backend/packages/shared/storage"
	"github.com/synthify/backend/packages/shared/util"
)

type API struct {
	Port                     string
	CORSAllowedOrigins       string
	GCSUploadURLBase         string
	InternalGCSUploadBase    string
	FirebaseProjectID        string
	FirebaseAuthEmulatorHost string
	WorkerBaseURL            string
}

type Worker struct {
	Port                     string
	GCSUploadURLBase         string
	FirebaseProjectID        string
	FirebaseAuthEmulatorHost string
}

type Store struct {
	DatabaseURL string
}

type LLM struct {
	GeminiAPIKey string
	GeminiModel  string
}

type Service struct {
	Mode string
}

func LoadAPI() API {
	uploadBase := mustBaseURL("GCS_UPLOAD_URL_BASE", get("GCS_UPLOAD_URL_BASE", "http://127.0.0.1:4443"))
	return API{
		Port:                     get("PORT", "8080"),
		CORSAllowedOrigins:       get("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173"),
		GCSUploadURLBase:         uploadBase,
		InternalGCSUploadBase:    mustBaseURL("INTERNAL_GCS_UPLOAD_URL_BASE", get("INTERNAL_GCS_UPLOAD_URL_BASE", uploadBase)),
		FirebaseProjectID:        os.Getenv("FIREBASE_PROJECT_ID"),
		FirebaseAuthEmulatorHost: os.Getenv("FIREBASE_AUTH_EMULATOR_HOST"),
		WorkerBaseURL:            os.Getenv("WORKER_BASE_URL"),
	}
}

func LoadWorker() Worker {
	return Worker{
		Port:                     get("PORT", "8080"),
		GCSUploadURLBase:         mustBaseURL("GCS_UPLOAD_URL_BASE", get("GCS_UPLOAD_URL_BASE", "http://127.0.0.1:4443")),
		FirebaseProjectID:        os.Getenv("FIREBASE_PROJECT_ID"),
		FirebaseAuthEmulatorHost: os.Getenv("FIREBASE_AUTH_EMULATOR_HOST"),
	}
}

func LoadStore() Store {
	return Store{DatabaseURL: os.Getenv("DATABASE_URL")}
}

func LoadLLM() LLM {
	return LLM{
		GeminiAPIKey: util.FirstNonEmpty(os.Getenv("GEMINI_API_KEY"), os.Getenv("GOOGLE_API_KEY")),
		GeminiModel:  get("GEMINI_MODEL", "gemini-3-flash-preview"),
	}
}

func LoadService() Service {
	return Service{Mode: get("SERVICE_MODE", "api")}
}

func (c LLM) Enabled() bool {
	return c.GeminiAPIKey != ""
}

func get(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func mustBaseURL(name, value string) string {
	if err := storage.ValidateBaseURL(value); err != nil {
		panic(fmt.Sprintf("%s is invalid: %v", name, err))
	}
	return value
}
