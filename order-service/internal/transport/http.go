package transport

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/internal/handler"
	"github.com/vasiliy-maslov/ecommerce-microservices/order-service/internal/order"
)

func NewRouter(dbConn *pgxpool.Pool) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	})

	repo := order.NewPostgresOrderRepository(dbConn)
	svc := order.NewOrderService(repo)
	h := handler.NewOrderHandler(svc)

	r.Post("/orders", h.CreateOrder)
	r.Get("/orders/{id}", h.GetOrderByID)

	return r
}
