package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"
	"github.com/Keyhole-Koro/SynthifyShared/config"
)

type contextKey string

const authUserContextKey contextKey = "auth_user"

type AuthUser struct {
	ID    string
	Email string
}

func CurrentUser(ctx context.Context) (AuthUser, bool) {
	user, ok := ctx.Value(authUserContextKey).(AuthUser)
	return user, ok
}

func WithAuth(projectID string, next http.Handler) http.Handler {
	// When running against the Firebase Auth emulator, accept X-E2E-User-Id
	// header instead of a real JWT so local dev works without Firebase login.
	if config.FirebaseAuthEmulatorEnabled() {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}
			userID := strings.TrimSpace(r.Header.Get("X-E2E-User-Id"))
			if userID == "" {
				log.Printf("Auth failure: missing X-E2E-User-Id header")
				http.Error(w, "missing e2e user header", http.StatusUnauthorized)
				return
			}
			user := AuthUser{
				ID:    userID,
				Email: strings.TrimSpace(r.Header.Get("X-E2E-User-Email")),
			}
			ctx := context.WithValue(r.Context(), authUserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	client, err := newFirebaseAuthClient(projectID)
	if err != nil {
		panic(err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("WithAuth: %s %s", r.Method, r.URL.Path)
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		token := bearerToken(authHeader)
		if token == "" {
			log.Printf("Auth failure: missing or invalid Authorization header: %q", authHeader)
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}

		idToken, err := client.VerifyIDToken(r.Context(), token)
		if err != nil {
			log.Printf("Auth failure: VerifyIDToken error: %v", err)
			http.Error(w, "invalid bearer token", http.StatusUnauthorized)
			return
		}

		email, _ := idToken.Claims["email"].(string)
		user := AuthUser{
			ID:    idToken.UID,
			Email: email,
		}
		ctx := context.WithValue(r.Context(), authUserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func newFirebaseAuthClient(projectID string) (*firebaseauth.Client, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, errors.New("FIREBASE_PROJECT_ID is required")
	}
	app, err := firebase.NewApp(context.Background(), &firebase.Config{ProjectID: projectID})
	if err != nil {
		return nil, err
	}
	return app.Auth(context.Background())
}

// ContextWithUser returns a copy of ctx carrying user.
// Intended for use in tests and integration utilities.
func ContextWithUser(ctx context.Context, user AuthUser) context.Context {
	return context.WithValue(ctx, authUserContextKey, user)
}

func bearerToken(header string) string {
	const prefix = "bearer "
	if !strings.HasPrefix(strings.ToLower(header), prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}
