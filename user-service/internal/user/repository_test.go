package user_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"github.com/vasiliy-maslov/ecommerce-microservices/user-service/internal/user"
)

var testDB *pgxpool.Pool

func TestMain(m *testing.M) {
	// --- Получаем параметры БД для тестов ---
	// Пытаемся читать из ENV с суффиксом _TEST, иначе используем дефолты для localhost
	dbHost := os.Getenv("DB_HOST_TEST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("DB_PORT_TEST")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbUser := os.Getenv("DB_USER_TEST")
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPassword := os.Getenv("DB_PASSWORD_TEST")
	if dbPassword == "" {
		dbPassword = "123456"
	}
	dbName := os.Getenv("DB_NAME_TEST")
	if dbName == "" {
		dbName = "ecommerce_db"
	}
	dbSSLMode := os.Getenv("DB_SSLMODE_TEST")
	if dbSSLMode == "" {
		dbSSLMode = "disable"
	}
	// --- КОНЕЦ Параметры БД ---

	// --- Установка соединения ---
	// Формируем строку подключения БЕЗ вызова config.NewConfig()
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s search_path=user_service",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	// Используем стандартные настройки пула для тестов
	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("host", dbHost).
			Str("port", dbPort).
			Str("user", dbUser).
			Str("dbname", dbName).
			Str("sslmode", dbSSLMode).
			Msg("Failed to connect to test database")
	}
	poolConfig.MaxConns = 5

	// Создаем контекст с таймаутом для подключения
	connectCtx, connectCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer connectCancel()

	testDB, err = pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		log.Fatal().Err(err).Str("db_host", dbHost).Str("db_port", dbPort).Msg("Failed to connect to test database")
	}

	// Пингуем с таймаутом
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err = testDB.Ping(pingCtx); err != nil {
		// Закрываем пул перед фатальной ошибкой, если он был создан
		if testDB != nil {
			testDB.Close()
		}
		log.Fatal().Err(err).Msg("Failed to ping test database")
	}
	log.Info().Msg("Test Database connection established.")
	// --- КОНЕЦ Установки соединения ---

	// Запуск тестов
	exitCode := m.Run()

	// Очистка
	if testDB != nil {
		testDB.Close()
		log.Info().Msg("TEST SETUP: Test Database connection closed.")
	}
	os.Exit(exitCode)
}

func truncateUsersTable(tb testing.TB, pool *pgxpool.Pool) {
	tb.Helper()
	_, err := pool.Exec(context.Background(), "TRUNCATE TABLE user_service.users RESTART IDENTITY CASCADE")
	require.NoError(tb, err, "failed to truncate users table")
}

func TestUserRepository_Create(t *testing.T) {
	repo := user.NewRepository(testDB)

	userID, _ := uuid.NewV4()
	testUser := user.User{
		ID:           userID,
		FirstName:    "Test",
		LastName:     "User",
		Email:        "test.create@example.com",
		PasswordHash: "hashed_password",
	}

	t.Cleanup(func() {
		truncateUsersTable(t, testDB)
	})

	createdID, err := repo.Create(context.Background(), &testUser)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, createdID)
	require.Equal(t, testUser.ID, createdID)
}

func TestUserRepository_Create_EmailExists(t *testing.T) {
	repo := user.NewRepository(testDB)

	t.Cleanup(func() {
		truncateUsersTable(t, testDB)
	})

	userID1, _ := uuid.NewV4()
	user1 := user.User{
		ID:           userID1,
		FirstName:    "Test",
		LastName:     "User",
		Email:        "test.create@example.com",
		PasswordHash: "hashed_password",
	}

	userID2, _ := uuid.NewV4()
	user2 := user.User{
		ID:           userID2,
		FirstName:    "Test",
		LastName:     "User",
		Email:        "test.create@example.com",
		PasswordHash: "hashed_password",
	}

	_, err := repo.Create(context.Background(), &user1)
	require.NoError(t, err)

	createdID, err := repo.Create(context.Background(), &user2)
	require.Error(t, err)
	require.ErrorIs(t, err, user.ErrEmailExists)
	require.Equal(t, uuid.Nil, createdID)
}

func TestUserRepository_GetById_Success(t *testing.T) {
	repo := user.NewRepository(testDB)

	t.Cleanup(func() {
		truncateUsersTable(t, testDB)
	})

	userID, _ := uuid.NewV4()
	user := user.User{
		ID:           userID,
		FirstName:    "Test",
		LastName:     "User",
		Email:        "test.create@example.com",
		PasswordHash: "hashed_password",
	}

	_, err := repo.Create(context.Background(), &user)
	require.NoError(t, err)

	foundUser, err := repo.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, foundUser)
	require.Equal(t, foundUser.ID, user.ID)
	require.Equal(t, foundUser.FirstName, user.FirstName)
	require.Equal(t, foundUser.LastName, user.LastName)
	require.Equal(t, foundUser.Email, user.Email)
	require.Equal(t, foundUser.PasswordHash, user.PasswordHash)
	require.False(t, foundUser.CreatedAt.IsZero())
	require.False(t, foundUser.UpdatedAt.IsZero())
}

func TestUserRepository_GetById_NotFound(t *testing.T) {
	repo := user.NewRepository(testDB)

	t.Cleanup(func() {
		truncateUsersTable(t, testDB)
	})

	nonExistentID, _ := uuid.NewV4()

	foundUser, err := repo.GetByID(context.Background(), nonExistentID)
	require.Error(t, err)
	require.ErrorIs(t, err, user.ErrNotFound)
	require.Nil(t, foundUser)
}

func TestUserRepository_GetByEmail_Success(t *testing.T) {
	repo := user.NewRepository(testDB)

	t.Cleanup(func() {
		truncateUsersTable(t, testDB)
	})

	userID, _ := uuid.NewV4()
	user := user.User{
		ID:           userID,
		FirstName:    "Test",
		LastName:     "User",
		Email:        "test.create@example.com",
		PasswordHash: "hashed_password",
	}

	_, err := repo.Create(context.Background(), &user)
	require.NoError(t, err)

	foundUser, err := repo.GetByEmail(context.Background(), user.Email)
	require.NoError(t, err)
	require.NotNil(t, foundUser)
	require.Equal(t, foundUser.ID, user.ID)
	require.Equal(t, foundUser.FirstName, user.FirstName)
	require.Equal(t, foundUser.LastName, user.LastName)
	require.Equal(t, foundUser.Email, user.Email)
	require.Equal(t, foundUser.PasswordHash, user.PasswordHash)
	require.False(t, foundUser.CreatedAt.IsZero())
	require.False(t, foundUser.UpdatedAt.IsZero())
}

func TestUserRepository_GetByEmail_NotFound(t *testing.T) {
	repo := user.NewRepository(testDB)

	t.Cleanup(func() {
		truncateUsersTable(t, testDB)
	})

	nonExistentEmail := "non.existent.mail@example.com"

	foundUser, err := repo.GetByEmail(context.Background(), nonExistentEmail)
	require.Error(t, err)
	require.ErrorIs(t, err, user.ErrNotFound)
	require.Nil(t, foundUser)
}

func TestUserRepository_Update_Success(t *testing.T) {
	repo := user.NewRepository(testDB)
	t.Cleanup(func() {
		truncateUsersTable(t, testDB)
	})

	// Arrange: Создаем исходного пользователя
	userID, _ := uuid.NewV4()
	initialUser := user.User{ // Переименуем для ясности
		ID:           userID,
		FirstName:    "Test",
		LastName:     "User",
		Email:        "test.update.success@example.com", // Уникальный email
		PasswordHash: "hashed_password",
	}
	_, err := repo.Create(context.Background(), &initialUser)
	require.NoError(t, err, "Failed to create initial user")

	// Arrange: Определяем, какие данные мы хотим обновить
	userToUpdate := user.User{ // Создаем объект с данными для обновления
		ID:           initialUser.ID,           // Используем тот же ID
		FirstName:    "Updated First",          // Новое имя
		LastName:     "Updated Last",           // Новая фамилия
		Email:        initialUser.Email,        // Email пока не меняем
		PasswordHash: initialUser.PasswordHash, // Пароль не меняем
		// CreatedAt и UpdatedAt не важны для передачи в Update
	}

	// Act: Выполняем обновление
	err = repo.Update(context.Background(), &userToUpdate)
	require.NoError(t, err, "Failed to update user")

	// Assert: Получаем обновленного пользователя из БД
	foundUser, err := repo.GetByID(context.Background(), initialUser.ID)
	require.NoError(t, err, "Failed to get user after update")
	require.NotNil(t, foundUser)

	// Assert: Сравниваем поля с ЯВНЫМИ ожидаемыми значениями
	require.Equal(t, initialUser.ID, foundUser.ID)                     // ID не должен меняться
	require.Equal(t, "Updated First", foundUser.FirstName)             // Проверяем новое имя
	require.Equal(t, "Updated Last", foundUser.LastName)               // Проверяем новую фамилию
	require.Equal(t, initialUser.Email, foundUser.Email)               // Email не менялся
	require.Equal(t, initialUser.PasswordHash, foundUser.PasswordHash) // Хеш не менялся

	// Assert: Проверяем время
	require.False(t, foundUser.CreatedAt.IsZero())
	require.False(t, foundUser.UpdatedAt.IsZero())
	// Сравниваем время обновленного пользователя из БД
	require.True(t, foundUser.UpdatedAt.After(foundUser.CreatedAt), "UpdatedAt should be after CreatedAt")
}

func TestUserRepository_Update_NotFound(t *testing.T) {
	repo := user.NewRepository(testDB)

	t.Cleanup(func() {
		truncateUsersTable(t, testDB)
	})

	userID, _ := uuid.NewV4()
	nonExistentUser := user.User{
		ID:           userID,
		FirstName:    "Test",
		LastName:     "User",
		Email:        "test.update@example.com",
		PasswordHash: "hashed_password",
	}

	err := repo.Update(context.Background(), &nonExistentUser)
	require.Error(t, err)
	require.ErrorIs(t, err, user.ErrNotFound)
}

func TestUserRepository_Update_EmailExists(t *testing.T) {
	repo := user.NewRepository(testDB)

	t.Cleanup(func() {
		truncateUsersTable(t, testDB)
	})

	userID, _ := uuid.NewV4()
	user1 := user.User{
		ID:           userID,
		FirstName:    "Test",
		LastName:     "User",
		Email:        "test.update1@example.com",
		PasswordHash: "hashed_password",
	}

	_, err := repo.Create(context.Background(), &user1)
	require.NoError(t, err)

	userID, _ = uuid.NewV4()
	user2 := user.User{
		ID:           userID,
		FirstName:    "Test",
		LastName:     "User",
		Email:        "test.update2@example.com",
		PasswordHash: "hashed_password",
	}

	_, err = repo.Create(context.Background(), &user2)
	require.NoError(t, err)

	user1.Email = user2.Email

	err = repo.Update(context.Background(), &user1)
	require.Error(t, err)
	require.ErrorIs(t, err, user.ErrEmailExists)
}

func TestUserRepository_Delete_Success(t *testing.T) {
	repo := user.NewRepository(testDB)

	t.Cleanup(func() {
		truncateUsersTable(t, testDB)
	})

	userID, _ := uuid.NewV4()
	createdUser := user.User{
		ID:           userID,
		FirstName:    "Test",
		LastName:     "User",
		Email:        "test.delete@example.com",
		PasswordHash: "hashed_password",
	}

	_, err := repo.Create(context.Background(), &createdUser)
	require.NoError(t, err)

	err = repo.Delete(context.Background(), createdUser.ID)
	require.NoError(t, err)

	foundUser, err := repo.GetByID(context.Background(), createdUser.ID)
	require.Error(t, err)
	require.ErrorIs(t, err, user.ErrNotFound)
	require.Nil(t, foundUser)
}

func TestUserRepository_Delete_NotFound(t *testing.T) {
	repo := user.NewRepository(testDB)

	t.Cleanup(func() {
		truncateUsersTable(t, testDB)
	})

	nonExistentID, _ := uuid.NewV4()

	err := repo.Delete(context.Background(), nonExistentID)
	require.Error(t, err)
	require.ErrorIs(t, err, user.ErrNotFound)
}
