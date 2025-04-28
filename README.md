# E-commerce Microservices

This project implements a microservices architecture for an e-commerce platform. The `order-service` manages customer orders, storing them in a PostgreSQL database.

## Requirements

- Go 1.21 or higher
- Docker and `docker-compose`
- PostgreSQL (provided via Docker)

## Installation

1. Clone the repository:

   ```bash
   git clone https://github.com/vasiliy-maslov/ecommerce-microservices.git
   cd ecommerce-microservices
   ```

2. Install Go dependencies:

   ```bash
   go mod download
   ```

## Configuration

1. Create a `.env` file in the `order-service` directory:

   ```bash
   echo -e "APP_PORT=8080\nDB_HOST=localhost\nDB_PORT=5432\nDB_USER=postgres\nDB_NAME=orders\nDB_PASSWORD=123456\nDB_SSLMODE=disable\nDB_MAX_CONNS=20\nDB_MIN_CONNS=2\nDB_MAX_CONN_LIFETIME=30m\nMIGRATIONS_PATH=migrations" > order-service/.env
   ```

2. Ensure `order-service/.env` is added to `.gitignore`:

   ```bash
   echo "order-service/.env" >> .gitignore
   ```

## Running the Service

All commands must be executed from the root directory (`ecommerce-microservices`).

1. Start the PostgreSQL database:

   ```bash
   docker-compose up -d
   ```

2. Run the `order-service`:

   ```bash
   cd order-service
   go run cmd/order-service/main.go
   ```

   Expected output:

   ```
   2025/04/18 10:56:58 Order service starting...
   2025/04/18 10:56:58 Connected to PostgreSQL
   2025/04/18 10:56:58 Migrations applied successfully
   2025/04/18 10:56:58 Starting server on :8080
   ```

3. To stop the database:

   ```bash
   docker-compose down
   ```

## Testing the API

Use Postman or any HTTP client to test the API at `http://localhost:8080`.

1. **Create an order**:

   - Method: `POST`

   - URL: `http://localhost:8080/orders`

   - Body (JSON):

     ```json
     {
         "id": "550e8400-e29b-41d4-a716-446655440000",
         "user_id": "123e4567-e89b-12d3-a456-426614174000",
         "total": 100.50,
         "status": "created",
         "created_at": "2025-04-16T12:00:00Z",
         "updated_at": "2025-04-16T12:00:00Z"
     }
     ```

   - Expected response: `201 Created`

2. **Get an order by ID**:

   - Method: `GET`
   - URL: `http://localhost:8080/orders/550e8400-e29b-41d4-a716-446655440000`
   - Expected response: `200 OK` with order details

3. **Get a non-existent order**:

   - Method: `GET`
   - URL: `http://localhost:8080/orders/999e8400-e29b-41d4-a716-446655440000`
   - Expected response: `404 Not Found` with body `"order not found"`

## Running Tests

Run unit tests for the `order-service`:

```bash
cd order-service
go test -v ./...
```

Expected output:

```
=== RUN   TestOrderService_CreateOrder
--- PASS: TestOrderService_CreateOrder (0.00s)
=== RUN   TestOrderService_GetOrderByID
--- PASS: TestOrderService_GetOrderByID (0.00s)
PASS
ok      github.com/vasiliy-maslov/ecommerce-microservices/order-service/services        0.002s
```

## Notes

- Always run commands from the root directory (`ecommerce-microservices`) to ensure correct paths for `order-service/.env` and `docker-compose.yml`.
- The `.env` file contains sensitive database credentials and must not be committed.