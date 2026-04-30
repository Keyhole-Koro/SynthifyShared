package config

import "os"

type API struct {
	Port                     string
	CORSAllowedOrigins       string
	GCSUploadURLBase         string
	InternalGCSUploadBase    string
	FirebaseProjectID        string
	FirebaseAuthEmulatorHost string
	WorkerBaseURL            string
	InternalWorkerToken      string
}

type Worker struct {
	Port                     string
	GCSUploadURLBase         string
	FirebaseProjectID        string
	FirebaseAuthEmulatorHost string
	InternalWorkerToken      string
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
	uploadBase := get("GCS_UPLOAD_URL_BASE", "http://localhost:4443/storage/v1/b/synthify-uploads/o")
	return API{
		Port:                     get("PORT", "8080"),
		CORSAllowedOrigins:       get("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173"),
		GCSUploadURLBase:         uploadBase,
		InternalGCSUploadBase:    get("INTERNAL_GCS_UPLOAD_URL_BASE", uploadBase),
		FirebaseProjectID:        os.Getenv("FIREBASE_PROJECT_ID"),
		FirebaseAuthEmulatorHost: os.Getenv("FIREBASE_AUTH_EMULATOR_HOST"),
		WorkerBaseURL:            os.Getenv("WORKER_BASE_URL"),
		InternalWorkerToken:      os.Getenv("INTERNAL_WORKER_TOKEN"),
	}
}

func LoadWorker() Worker {
	return Worker{
		Port:                     get("PORT", "8080"),
		GCSUploadURLBase:         get("GCS_UPLOAD_URL_BASE", "http://localhost:4443/storage/v1/b/synthify-uploads/o"),
		FirebaseProjectID:        os.Getenv("FIREBASE_PROJECT_ID"),
		FirebaseAuthEmulatorHost: os.Getenv("FIREBASE_AUTH_EMULATOR_HOST"),
		InternalWorkerToken:      os.Getenv("INTERNAL_WORKER_TOKEN"),
	}
}

func LoadStore() Store {
	return Store{DatabaseURL: os.Getenv("DATABASE_URL")}
}

func LoadLLM() LLM {
	return LLM{
		GeminiAPIKey: firstNonEmpty(os.Getenv("GEMINI_API_KEY"), os.Getenv("GOOGLE_API_KEY")),
		GeminiModel:  get("GEMINI_MODEL", "gemini-3-flash-preview"),
	}
}

func LoadService() Service {
	return Service{Mode: get("SERVICE_MODE", "api")}
}

func (c LLM) Enabled() bool {
	return c.GeminiAPIKey != ""
}

func FirebaseAuthEmulatorEnabled() bool {
	return os.Getenv("FIREBASE_AUTH_EMULATOR_HOST") != ""
}

func get(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
