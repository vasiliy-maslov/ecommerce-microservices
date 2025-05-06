package user_test

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vasiliy-maslov/ecommerce-microservices/user-service/internal/user"
	"golang.org/x/crypto/bcrypt"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, u *user.User) (uuid.UUID, error) {
	args := m.Called(ctx, u)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, u *user.User) error {
	args := m.Called(ctx, u)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestUserService_CreateUser_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockUserRepository)      // Создаем экземпляр мока
	userService := user.NewService(mockRepo) // Внедряем мок в сервис

	testUser := &user.User{
		FirstName:    "Test",
		LastName:     "User",
		Email:        "duplicate@example.com", // Не важно для этого теста, но зададим
		PasswordHash: "somepassword",
	}
	expectedID := uuid.Must(uuid.NewV4()) // ID, который мы ХОТИМ, чтобы мок вернул

	// НАСТРОЙКА МОКА:
	// Говорим: "Когда будет вызван метод Create с ЛЮБЫМ контекстом
	// и указателем на ЛЮБОЙ user.User, ТО ВЕРНУТЬ expectedID и nil (нет ошибки)"
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*user.User")).
		Return(expectedID, nil).
		Once() // Ожидаем, что вызов будет только один раз

	// Act
	createdUser, err := userService.CreateUser(context.Background(), testUser)

	require.NoError(t, err)
	require.NotNil(t, createdUser)
	require.Equal(t, expectedID, createdUser.ID) // Проверяем, что сервис вернул то, что дал мок

	rawPassword := "somepassword"

	err = bcrypt.CompareHashAndPassword([]byte(createdUser.PasswordHash), []byte(rawPassword))
	require.NoError(t, err, "Password hash does not match raw password")
	require.NotEqual(t, rawPassword, createdUser.PasswordHash, "Password should be hashed, not raw")

	// Проверяем, что все ожидания мока были выполнены
	mockRepo.AssertExpectations(t)
}

func TestUserService_CreateUser_EmailExists(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := user.NewService(mockRepo)

	testUser := user.User{
		FirstName:    "Test",
		LastName:     "User",
		Email:        "duplicate@example.com", // Не важно для этого теста, но зададим
		PasswordHash: "somepassword",          // Главное - не пустой!
	}

	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*user.User")).
		Return(uuid.Nil, user.ErrEmailExists).
		Once()

	createdUser, err := userService.CreateUser(context.Background(), &testUser)
	require.Error(t, err)
	require.ErrorIs(t, err, user.ErrEmailExists)
	require.Nil(t, createdUser)
	mockRepo.AssertExpectations(t)
}

func TestUserService_GetUserByID_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := user.NewService(mockRepo)

	userID := uuid.Must(uuid.NewV4())

	expectedUser := user.User{
		ID:           userID,
		FirstName:    "Test",
		LastName:     "User",
		Email:        "getbyid@example.com",
		PasswordHash: "hashed_password_from_repo",
		CreatedAt:    time.Now().Add(-time.Hour),
		UpdatedAt:    time.Now(),
	}

	mockRepo.On("GetByID", mock.Anything, userID).
		Return(&expectedUser, nil).
		Once()

	foundUser, err := userService.GetUserByID(context.Background(), userID)

	require.NoError(t, err)
	require.NotNil(t, foundUser)
	diff := cmp.Diff(expectedUser, *foundUser)
	require.Empty(t, diff)
	mockRepo.AssertExpectations(t)
}

func TestUserService_GetUserByID_NotFound(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := user.NewService(mockRepo)

	userID := uuid.Must(uuid.NewV4())

	mockRepo.On("GetByID", mock.Anything, userID).
		Return(nil, user.ErrNotFound).
		Once()

	foundUser, err := userService.GetUserByID(context.Background(), userID)
	require.Error(t, err)
	require.ErrorIs(t, err, user.ErrNotFound)
	require.Nil(t, foundUser)
	mockRepo.AssertExpectations(t)
}

func TestUserService_GetUserByEmail_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := user.NewService(mockRepo)

	userID := uuid.Must(uuid.NewV4())
	userEmail := "getbyid@example.com"

	expectedUser := user.User{
		ID:           userID,
		FirstName:    "Test",
		LastName:     "User",
		Email:        userEmail,
		PasswordHash: "hashed_password_from_repo",
		CreatedAt:    time.Now().Add(-time.Hour),
		UpdatedAt:    time.Now(),
	}

	mockRepo.On("GetByEmail", mock.Anything, userEmail).
		Return(&expectedUser, nil).
		Once()

	foundUser, err := userService.GetUserByEmail(context.Background(), userEmail)

	require.NoError(t, err)
	require.NotNil(t, foundUser)
	diff := cmp.Diff(expectedUser, *foundUser)
	require.Empty(t, diff)
	mockRepo.AssertExpectations(t)
}

func TestUserService_GetUserByEmail_NotFound(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := user.NewService(mockRepo)

	userEmail := "getbyid@example.com"

	mockRepo.On("GetByEmail", mock.Anything, userEmail).
		Return(nil, user.ErrNotFound).
		Once()

	foundUser, err := userService.GetUserByEmail(context.Background(), userEmail)
	require.Error(t, err)
	require.ErrorIs(t, err, user.ErrNotFound)
	require.Nil(t, foundUser)
	mockRepo.AssertExpectations(t)
}

func TestUserService_UpdateUser_Success_NoPasswordChange(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := user.NewService(mockRepo)

	userID := uuid.Must(uuid.NewV4())

	userToUpdate := user.User{
		ID:        userID,
		FirstName: "Test_Updated",
		LastName:  "User_Updated",
		Email:     "getbyid@example.com",
	}

	mockRepo.On("Update", mock.Anything, &userToUpdate).
		Return(nil).
		Once()

	err := userService.UpdateUser(context.Background(), &userToUpdate)
	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUserService_UpdateUser_Success_WithPasswordChange(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := user.NewService(mockRepo)

	userID := uuid.Must(uuid.NewV4())

	rawPassword := "newpassword123"
	userToUpdate := user.User{
		ID:           userID,
		FirstName:    "Test_Updated",
		LastName:     "User_Updated",
		Email:        "getbyid@example.com",
		PasswordHash: rawPassword,
	}

	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(u *user.User) bool {
		return u.ID == userToUpdate.ID &&
			u.PasswordHash != rawPassword &&
			u.PasswordHash != "" &&
			bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(rawPassword)) == nil
	})).
		Return(nil).
		Once()

	err := userService.UpdateUser(context.Background(), &userToUpdate)
	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUserService_UpdateUser_EmailExists(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := user.NewService(mockRepo)

	userID := uuid.Must(uuid.NewV4())

	userToUpdate := user.User{
		ID:           userID,
		FirstName:    "Test",
		LastName:     "User",
		Email:        "getbyid@example.com",
		PasswordHash: "newpassword",
	}

	mockRepo.On("Update", mock.Anything, &userToUpdate).
		Return(user.ErrEmailExists).
		Once()

	err := userService.UpdateUser(context.Background(), &userToUpdate)
	require.Error(t, err)
	require.ErrorIs(t, err, user.ErrEmailExists)
	mockRepo.AssertExpectations(t)
}

func TestUserService_UpdateUser_NotFound(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := user.NewService(mockRepo)

	userID := uuid.Must(uuid.NewV4())

	userToUpdate := user.User{
		ID:           userID,
		FirstName:    "Test",
		LastName:     "User",
		Email:        "getbyid@example.com",
		PasswordHash: "newpassword",
	}

	mockRepo.On("Update", mock.Anything, &userToUpdate).
		Return(user.ErrNotFound).
		Once()

	err := userService.UpdateUser(context.Background(), &userToUpdate)
	require.Error(t, err)
	require.ErrorIs(t, err, user.ErrNotFound)
	mockRepo.AssertExpectations(t)
}

func TestUserService_DeleteUser_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := user.NewService(mockRepo)

	userID := uuid.Must(uuid.NewV4())

	mockRepo.On("Delete", mock.Anything, userID).
		Return(nil).
		Once()

	err := userService.DeleteUser(context.Background(), userID)
	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUserService_DeleteUser_NotFound(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := user.NewService(mockRepo)

	userID := uuid.Must(uuid.NewV4())

	mockRepo.On("Delete", mock.Anything, userID).
		Return(user.ErrNotFound).
		Once()

	err := userService.DeleteUser(context.Background(), userID)
	require.Error(t, err)
	require.ErrorIs(t, err, user.ErrNotFound)
	mockRepo.AssertExpectations(t)
}
