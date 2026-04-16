package httpapi

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type UserIDContextKey struct{}

func NewRouter(productHandler *ProductHandler) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: log.Default()}))
	r.Route("/product", func(r chi.Router) {
		r.Use(FakeAuthMiddleware)
		r.Post("/", productHandler.Create)
		r.Get("/", productHandler.List)
		r.Get("/{id}", productHandler.Get)
	})
	return r
}

func FakeAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
		if err != nil {
			log.Println("failed to parse user id from header:", err)
			http.Error(w, "Invalid user id", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), UserIDContextKey{}, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func MustGetUserID(ctx context.Context) int64 {
	userID, ok := ctx.Value(UserIDContextKey{}).(int64)
	if !ok {
		panic("user id not found in context")
	}
	return userID
}
