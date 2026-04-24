package httpapi

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type userIDContextKey struct{}

func NewRouter(orderHandler *OrderHandler) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: log.Default()}))
	r.Route("/orders", func(r chi.Router) {
		r.Use(FakeAuthMiddleware)
		r.Post("/create", orderHandler.Create)
		r.Post("/cancel", orderHandler.Cancel)
		r.Post("/complete", orderHandler.Complete)
		r.Get("/", orderHandler.Get)
	})
	return r
}

func FakeAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
		if err != nil {
			log.Println("failed to parse user id from header:", err)
			http.Error(w, "invalid user id", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), userIDContextKey{}, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func MustGetUserID(ctx context.Context) int64 {
	userID, ok := ctx.Value(userIDContextKey{}).(int64)
	if !ok {
		panic("user id not found in context")
	}
	return userID
}
