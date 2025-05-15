package http_test

import (
	"bytes" // Для создания io.Reader из строки JSON
	"context"
	"encoding/json"
	"net/http"          // Для http.StatusOK и др.
	"net/http/httptest" // Для ResponseRecorder и NewRequest
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gofrs/uuid"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert" // Будем использовать assert для проверок
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	userHandler "github.com/vasiliy-maslov/ecommerce-microservices/user-service/internal/handler/http"
	"github.com/vasiliy-maslov/ecommerce-microservices/user-service/internal/user"
)

type MockUserService struct {
	mock.Mock
}

// Реализуем все методы интерфейса user.Service
func (m *MockUserService) CreateUser(ctx context.Context, u *user.User) (*user.User, error) {
	args := m.Called(ctx, u)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) GetUserByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) GetUserByEmail(ctx context.Context, email string) (*user.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) UpdateUser(ctx context.Context, u *user.User) error {
	args := m.Called(ctx, u)
	return args.Error(0)
}

func (m *MockUserService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestUserHandler_handleCreateUser_Success(t *testing.T) {
	mockService := new(MockUserService)
	handler := userHandler.NewUserHandler(mockService)

	requestDTO := userHandler.CreateUserRequest{
		FirstName: "Test",
		LastName:  "Uesr",
		Email:     "testcreate@example.com",
		Password:  "password123",
	}

	mockServiceResponseUser := user.User{
		ID:           uuid.Must(uuid.NewV4()),
		FirstName:    requestDTO.FirstName,
		LastName:     requestDTO.LastName,
		Email:        requestDTO.Email,
		PasswordHash: "hashed_password_from_service",
		CreatedAt:    time.Now().Truncate(time.Second),
		UpdatedAt:    time.Now().Truncate(time.Second),
	}

	mockService.On("CreateUser", mock.Anything, mock.MatchedBy(func(u *user.User) bool {
		return u.FirstName == requestDTO.FirstName &&
			u.LastName == requestDTO.LastName &&
			u.Email == requestDTO.Email &&
			u.PasswordHash == requestDTO.Password
	})).Return(&mockServiceResponseUser, nil).Once()

	jsonBody, err := json.Marshal(requestDTO)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code)

	var actualResponse userHandler.UserResponse
	err = json.NewDecoder(rr.Body).Decode(&actualResponse)
	require.NoError(t, err, "Failed to decode response body")

	assert.Equal(t, mockServiceResponseUser.ID, actualResponse.ID, "ID mismatch")
	assert.Equal(t, mockServiceResponseUser.FirstName, actualResponse.FirstName, "FirstName mismatch")
	assert.Equal(t, mockServiceResponseUser.LastName, actualResponse.LastName, "LastName mismatch")
	assert.Equal(t, mockServiceResponseUser.Email, actualResponse.Email, "Email mismatch")
	assert.WithinDuration(t, mockServiceResponseUser.CreatedAt, actualResponse.CreatedAt, time.Second, "CreatedAt mismatch")
	assert.WithinDuration(t, mockServiceResponseUser.UpdatedAt, actualResponse.UpdatedAt, time.Second, "UpdatedAt mismatch")
	mockService.AssertExpectations(t)
}

func TestUserHandler_handleCreateUser_EmailExists(t *testing.T) {
	mockService := new(MockUserService)
	handler := userHandler.NewUserHandler(mockService)

	requestDTO := userHandler.CreateUserRequest{
		FirstName: "Test",
		LastName:  "Uesr",
		Email:     "exists@example.com",
		Password:  "password123",
	}

	mockService.On("CreateUser", mock.Anything, mock.AnythingOfType("*user.User")).
		Return(nil, user.ErrEmailExists).
		Once()

	jsonBody, err := json.Marshal(requestDTO)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusConflict, rr.Code)

	var errorResponse map[string]string
	err = json.NewDecoder(rr.Body).Decode(&errorResponse)
	require.NoError(t, err, "Failed to decode error response body")
	assert.Contains(t, errorResponse["error"], "Email already exists", "Error message mismatch")
	mockService.AssertExpectations(t)
}

func TestUserHandler_handleCreateUser_InvalidJSON(t *testing.T) {
	mockService := new(MockUserService)
	handler := userHandler.NewUserHandler(mockService)

	invalidJsonString := `{"first_name": "Test", "last_name": "User", "email": "invalid@example.com" "password": "pass}`

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(invalidJsonString))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)

	var errorResponse map[string]string
	err := json.NewDecoder(rr.Body).Decode(&errorResponse)
	require.NoError(t, err, "Failed to decode error response body for invalid JSON test")
	assert.Contains(t, errorResponse["error"], "Invalid request payload", "Error message for invalid JSON mismatch")
	mockService.AssertNotCalled(t, "CreateUser", mock.Anything, mock.AnythingOfType("*user.User"))
}

func TestUserHandler_handleCreateUser_ValidationError(t *testing.T) {
	mockService := new(MockUserService)
	handler := userHandler.NewUserHandler(mockService)

	requestUser := userHandler.CreateUserRequest{
		FirstName: "J",
		Email:     "incorrect-email",
		Password:  "123456",
	}

	jsonBody, err := json.Marshal(requestUser)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)

	var errorResponse map[string]string
	err = json.NewDecoder(rr.Body).Decode(&errorResponse)
	require.NoError(t, err, "Failed to decode error response body for validation error test")

	actualErrorMessage, ok := errorResponse["error"]
	require.True(t, ok, "Error response should contain an 'error' field")

	assert.Contains(t, actualErrorMessage, "Field 'FirstName' must be at least 2 characters long")
	assert.Contains(t, actualErrorMessage, "Field 'LastName' is required")
	assert.Contains(t, actualErrorMessage, "Field 'Email' must be a valid email address")
	assert.Contains(t, actualErrorMessage, "Field 'Password' must be at least 8 characters long")

	mockService.AssertNotCalled(t, "CreateUser", mock.Anything, mock.AnythingOfType("*user.User"))
}

func TestUserHandler_handleUpdateUser_InvalidJSON(t *testing.T) {
	mockService := new(MockUserService)
	handler := userHandler.NewUserHandler(mockService)

	userID := uuid.Must(uuid.NewV4())

	invalidJsonString := `{"first_name": "Test", "last_name": "User", "email": "invalid@example.com" "password": "pass}`

	req := httptest.NewRequest(http.MethodPut, "/users/"+userID.String(), bytes.NewBufferString(invalidJsonString))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)

	var errorResponse map[string]string
	err := json.NewDecoder(rr.Body).Decode(&errorResponse)
	require.NoError(t, err, "Failed to decode error response body for invalid JSON test")
	assert.Contains(t, errorResponse["error"], "Invalid request payload", "Error message for invalid JSON mismatch")
	mockService.AssertNotCalled(t, "UpdateUser", mock.Anything, mock.AnythingOfType("*user.User"))
}

func TestUserHandler_handleUpdateUser_ValidationError(t *testing.T) {
	mockService := new(MockUserService)
	handler := userHandler.NewUserHandler(mockService)

	userID := uuid.Must(uuid.NewV4())

	requestUser := userHandler.UpdateUserRequest{
		FirstName: "U",
		LastName:  "",
		Email:     "not-valid",
	}

	reqBody, err := json.Marshal(requestUser)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/users/"+userID.String(), bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)

	var errorResponse map[string]string
	err = json.NewDecoder(rr.Body).Decode(&errorResponse)
	require.NoError(t, err, "Failed to decode error response body for validation error test")

	actualErrorMessage, ok := errorResponse["error"]
	require.True(t, ok, "Error response should contain an 'error' field")

	assert.Contains(t, actualErrorMessage, "Field 'FirstName' must be at least 2 characters long")
	assert.Contains(t, actualErrorMessage, "Field 'LastName' is required")
	assert.Contains(t, actualErrorMessage, "Field 'Email' must be a valid email address")

	mockService.AssertNotCalled(t, "UpdateUser", mock.Anything, mock.AnythingOfType("*user.User"))
}

func TestUserHandler_handleUpdateUser_NotFound(t *testing.T) {
	mockService := new(MockUserService)
	handler := userHandler.NewUserHandler(mockService)

	userID := uuid.Must(uuid.NewV4())

	requestUser := userHandler.UpdateUserRequest{
		FirstName: "User",
		LastName:  "Test",
		Email:     "mail@example.com",
	}

	mockService.On("UpdateUser", mock.Anything, mock.MatchedBy(func(u *user.User) bool {
		return u.ID == userID
	})).Return(user.ErrNotFound).Once()

	reqBody, err := json.Marshal(requestUser)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/users/"+userID.String(), bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusNotFound, rr.Code)

	var errorResponse map[string]string
	err = json.NewDecoder(rr.Body).Decode(&errorResponse)
	require.NoError(t, err, "Failed to decode error response body for validation error test")

	assert.Contains(t, errorResponse["error"], "User not found")
	mockService.AssertExpectations(t)
}

func TestUserHandler_handleUpdateUser_EmailExists(t *testing.T) {
	mockService := new(MockUserService)
	handler := userHandler.NewUserHandler(mockService)

	userID := uuid.Must(uuid.NewV4())

	requestUser := userHandler.UpdateUserRequest{
		FirstName: "User",
		LastName:  "Test",
		Email:     "exists_mail@example.com",
	}

	mockService.On("UpdateUser", mock.Anything, mock.MatchedBy(func(u *user.User) bool {
		return u.ID == userID
	})).Return(user.ErrEmailExists).Once()

	reqBody, err := json.Marshal(requestUser)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/users/"+userID.String(), bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusConflict, rr.Code)

	var errorResponse map[string]string
	err = json.NewDecoder(rr.Body).Decode(&errorResponse)
	require.NoError(t, err, "Failed to decode error response body for validation error test")

	assert.Contains(t, errorResponse["error"], "Email already exists")
	mockService.AssertExpectations(t)
}

func TestUserHandler_handleUpdateUser_InvalidUUID(t *testing.T) {
	mockService := new(MockUserService)
	handler := userHandler.NewUserHandler(mockService)

	invalidID := "not-a-uuid"

	requestUser := userHandler.UpdateUserRequest{
		FirstName: "User",
		LastName:  "Test",
		Email:     "mail@example.com",
	}
	reqBody, err := json.Marshal(requestUser)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/users/"+invalidID, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)

	var errorResponse map[string]string
	err = json.NewDecoder(rr.Body).Decode(&errorResponse)
	require.NoError(t, err)

	assert.Contains(t, errorResponse["error"], "Invalid id parameter")
	mockService.AssertNotCalled(t, "UpdateUser")
}

func TestUserHandler_handleUpdateUser_Success_NoPasswordChange(t *testing.T) {
	mockService := new(MockUserService)
	handler := userHandler.NewUserHandler(mockService)
	userID := uuid.Must(uuid.NewV4())

	requestUser := userHandler.UpdateUserRequest{
		FirstName: "User",
		LastName:  "Test",
		Email:     "mail@example.com",
		Password:  nil,
	}

	mockService.On("UpdateUser", mock.Anything, mock.MatchedBy(func(u *user.User) bool {
		return u.ID == userID &&
			u.FirstName == requestUser.FirstName &&
			u.LastName == requestUser.LastName &&
			u.Email == requestUser.Email &&
			u.PasswordHash == ""
	})).
		Return(nil).
		Once()

	reqBody, err := json.Marshal(requestUser)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/users/"+userID.String(), bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusNoContent, rr.Code)

	mockService.AssertExpectations(t)
}

func TestUserHandler_handleUpdateUser_Success_WithPasswordChange(t *testing.T) {
	mockService := new(MockUserService)
	handler := userHandler.NewUserHandler(mockService)
	userID := uuid.Must(uuid.NewV4())

	rawPassword := "new-password"

	requestDTO := userHandler.UpdateUserRequest{
		FirstName: "User",
		LastName:  "Test",
		Email:     "mail@example.com",
		Password:  &rawPassword,
	}

	mockService.On("UpdateUser", mock.Anything, mock.MatchedBy(func(u *user.User) bool {
		return u.ID == userID &&
			u.FirstName == requestDTO.FirstName &&
			u.LastName == requestDTO.LastName &&
			u.Email == requestDTO.Email &&
			u.PasswordHash == rawPassword
	})).
		Return(nil).
		Once()

	reqBody, err := json.Marshal(requestDTO)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/users/"+userID.String(), bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusNoContent, rr.Code)

	mockService.AssertExpectations(t)
}

func TestUserHandler_handleDeleteUser_Success(t *testing.T) {
	mockService := new(MockUserService)
	handler := userHandler.NewUserHandler(mockService)

	userID := uuid.Must(uuid.NewV4())

	mockService.On("DeleteUser", mock.Anything, userID).
		Return(nil).
		Once()

	req := httptest.NewRequest(http.MethodDelete, "/users/"+userID.String(), nil)
	rr := httptest.NewRecorder()

	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusNoContent, rr.Code)

	mockService.AssertExpectations(t)
}

func TestUserHandler_handleDeleteUser_NotFound(t *testing.T) {
	mockService := new(MockUserService)
	handler := userHandler.NewUserHandler(mockService)

	userID := uuid.Must(uuid.NewV4())

	mockService.On("DeleteUser", mock.Anything, userID).
		Return(user.ErrNotFound).
		Once()

	req := httptest.NewRequest(http.MethodDelete, "/users/"+userID.String(), nil)
	rr := httptest.NewRecorder()

	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusNotFound, rr.Code)

	mockService.AssertExpectations(t)
}

func TestUserHandler_handleDeleteUser_InvalidUUID(t *testing.T) {
	mockService := new(MockUserService)
	handler := userHandler.NewUserHandler(mockService)
	invalidID := "invalid_id"

	req := httptest.NewRequest(http.MethodDelete, "/users/"+invalidID, nil)
	rr := httptest.NewRecorder()

	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)

	mockService.AssertNotCalled(t, "DeleteUser", mock.Anything, mock.AnythingOfType("uuid.UUID"))
}

func TestUserHandler_handleGetUserByID_Success(t *testing.T) {
	mockService := new(MockUserService)
	handler := userHandler.NewUserHandler(mockService)
	userID := uuid.Must(uuid.NewV4())

	mockServiceReturnUser := user.User{
		ID:           userID,
		FirstName:    "User",
		LastName:     "Test",
		Email:        "mail@example.com",
		PasswordHash: "password_hash",
		CreatedAt:    time.Now().Add(-2 * time.Hour).Truncate(time.Second),
		UpdatedAt:    time.Now().Add(-1 * time.Hour).Truncate(time.Second),
	}

	mockService.On("GetUserByID", mock.Anything, userID).
		Return(&mockServiceReturnUser, nil).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/users/"+userID.String(), nil)
	rr := httptest.NewRecorder()

	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var actualResponse userHandler.UserResponse
	err := json.NewDecoder(rr.Body).Decode(&actualResponse)
	require.NoError(t, err)

	expectedResponse := userHandler.UserResponse{
		ID:        mockServiceReturnUser.ID,
		FirstName: mockServiceReturnUser.FirstName,
		LastName:  mockServiceReturnUser.LastName,
		Email:     mockServiceReturnUser.Email,
		CreatedAt: mockServiceReturnUser.CreatedAt,
		UpdatedAt: mockServiceReturnUser.UpdatedAt,
	}

	diff := cmp.Diff(expectedResponse, actualResponse)
	require.Empty(t, diff, "UserResponse mismatch (-expected +got):\n%s", diff)

	mockService.AssertExpectations(t)
}

func TestUserHandler_handleGetUserByID_NotFound(t *testing.T) {
	mockService := new(MockUserService)
	handler := userHandler.NewUserHandler(mockService)
	userID := uuid.Must(uuid.NewV4())

	mockService.On("GetUserByID", mock.Anything, userID).
		Return(nil, user.ErrNotFound).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/users/"+userID.String(), nil)
	rr := httptest.NewRecorder()

	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusNotFound, rr.Code)

	var errorResponse map[string]string
	err := json.NewDecoder(rr.Body).Decode(&errorResponse)
	require.NoError(t, err)
	assert.Contains(t, errorResponse["error"], "User not found")

	mockService.AssertExpectations(t)
}

func TestUserHandler_handleGetUserByID_InvalidUUID(t *testing.T) {
	mockService := new(MockUserService)
	handler := userHandler.NewUserHandler(mockService)
	invalidID := "not-a-uuid"

	req := httptest.NewRequest(http.MethodGet, "/users/"+invalidID, nil)
	rr := httptest.NewRecorder()

	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)

	var errorResponse map[string]string
	err := json.NewDecoder(rr.Body).Decode(&errorResponse)
	require.NoError(t, err)
	assert.Contains(t, errorResponse["error"], "Invalid id parameter")

	mockService.AssertNotCalled(t, "GetUserByID", mock.Anything, mock.AnythingOfType("uuid.UUID"))
}

func TestUserHandler_handleGetUserByEmail_Success(t *testing.T) {
	mockService := new(MockUserService)
	handler := userHandler.NewUserHandler(mockService)
	userID := uuid.Must(uuid.NewV4())
	userEmail := "mail@example.com"

	mockServiceReturnUser := user.User{
		ID:           userID,
		FirstName:    "User",
		LastName:     "Test",
		Email:        userEmail,
		PasswordHash: "password_hash",
		CreatedAt:    time.Now().Add(-2 * time.Hour).Truncate(time.Second),
		UpdatedAt:    time.Now().Add(-1 * time.Hour).Truncate(time.Second),
	}

	mockService.On("GetUserByEmail", mock.Anything, userEmail).
		Return(&mockServiceReturnUser, nil).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/users/email/"+userEmail, nil)
	rr := httptest.NewRecorder()

	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var actualResponse userHandler.UserResponse
	err := json.NewDecoder(rr.Body).Decode(&actualResponse)
	require.NoError(t, err)

	expectedResponse := userHandler.UserResponse{
		ID:        mockServiceReturnUser.ID,
		FirstName: mockServiceReturnUser.FirstName,
		LastName:  mockServiceReturnUser.LastName,
		Email:     mockServiceReturnUser.Email,
		CreatedAt: mockServiceReturnUser.CreatedAt,
		UpdatedAt: mockServiceReturnUser.UpdatedAt,
	}

	diff := cmp.Diff(expectedResponse, actualResponse)
	require.Empty(t, diff, "UserResponse mismatch (-expected +got):\n%s", diff)

	mockService.AssertExpectations(t)
}

func TestUserHandler_handleGetUserByEmail_NotFound(t *testing.T) {
	mockService := new(MockUserService)
	handler := userHandler.NewUserHandler(mockService)
	userEmail := "mail@example.com"

	mockService.On("GetUserByEmail", mock.Anything, userEmail).
		Return(nil, user.ErrNotFound).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/users/email/"+userEmail, nil)
	rr := httptest.NewRecorder()

	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusNotFound, rr.Code)

	var errorResponse map[string]string
	err := json.NewDecoder(rr.Body).Decode(&errorResponse)
	require.NoError(t, err)
	assert.Contains(t, errorResponse["error"], "User not found")

	mockService.AssertExpectations(t)
}

func TestUserHandler_handleGetUserByEmail_EmptyEmailAsNotFound(t *testing.T) {
	mockService := new(MockUserService)
	handler := userHandler.NewUserHandler(mockService)
	emptyEmail := ""

	req := httptest.NewRequest(http.MethodGet, "/users/email/"+emptyEmail, nil)
	rr := httptest.NewRecorder()
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusNotFound, rr.Code)

	mockService.AssertNotCalled(t, "GetUserByEmail")
}
