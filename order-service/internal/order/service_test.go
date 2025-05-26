package order

import (
	"context"
	"errors"
	"fmt"
	"math"
	"testing"
	"time"

	// Может понадобиться для сравнения полей Order
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockOrderRepository является моком для OrderRepository
type MockOrderRepository struct {
	mock.Mock
}

func (m *MockOrderRepository) CreateOrder(ctx context.Context, order *Order) (uuid.UUID, error) {
	args := m.Called(ctx, order)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockOrderRepository) GetOrderByID(ctx context.Context, id uuid.UUID) (*Order, error) {
	args := m.Called(ctx, id)

	var orderToReturn *Order
	if firstArg := args.Get(0); firstArg != nil {
		orderToReturn = firstArg.(*Order)
	}

	return orderToReturn, args.Error(1)
}

func (m *MockOrderRepository) GetOrdersByUserID(ctx context.Context, userID uuid.UUID) ([]Order, error) {
	args := m.Called(ctx, userID)

	var ordersToReturn []Order
	if args.Get(0) != nil {
		ordersToReturn = args.Get(0).([]Order)
	}

	return ordersToReturn, args.Error(1)
}

func (m *MockOrderRepository) UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, newStatus OrderStatus) error {
	args := m.Called(ctx, orderID, newStatus)
	return args.Error(0)
}

func TestService_CreateOrder_Success(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	orderService := NewService(mockRepo) // Используем твою фабричную функцию

	ctx := context.Background()

	userID := uuid.Must(uuid.NewV4())
	productID1 := uuid.Must(uuid.NewV4())
	orderInput := &Order{
		UserID: userID,
		OrderItems: []OrderItem{
			{ProductID: productID1, Quantity: 1, PricePerUnit: 10.0},
		},
		ShippingAddressText: "Test Address",
	}

	expectedOrderID := uuid.Must(uuid.NewV4())
	expectedOrderItemID1 := uuid.Must(uuid.NewV4())
	expectedTotalAmount := 10.0
	now := time.Now().UTC().Truncate(time.Microsecond)

	mockRepo.On("CreateOrder", ctx, mock.MatchedBy(func(argOrder *Order) bool {
		return argOrder.ID == uuid.Nil &&
			argOrder.Status == StatusNew &&
			math.Abs(argOrder.TotalAmount-expectedTotalAmount) < 0.001 &&
			argOrder.OrderItems[0].ID == uuid.Nil &&
			argOrder.OrderItems[0].OrderID == uuid.Nil
	})).
		Return(expectedOrderID, nil).
		Run(func(args mock.Arguments) {
			orderArg := args.Get(1).(*Order)
			orderArg.ID = expectedOrderID
			orderArg.CreatedAt = now
			orderArg.UpdatedAt = now
			if len(orderArg.OrderItems) > 0 { // Защита, если вдруг нет OrderItems
				orderArg.OrderItems[0].ID = expectedOrderItemID1
				orderArg.OrderItems[0].OrderID = expectedOrderID // Связываем позицию с ID заказа
				orderArg.OrderItems[0].CreatedAt = now
				orderArg.OrderItems[0].UpdatedAt = now
			}
		})

	createdOrder, err := orderService.CreateOrder(ctx, orderInput)

	require.NoError(t, err)
	require.NotNil(t, createdOrder)

	assert.Equal(t, expectedOrderID, createdOrder.ID)
	assert.Equal(t, userID, createdOrder.UserID)                            // Сравниваем с исходным userID
	assert.Equal(t, StatusNew, createdOrder.Status)                         // Статус должен быть StatusNew
	assert.InDelta(t, expectedTotalAmount, createdOrder.TotalAmount, 0.001) // Используем InDelta для float
	assert.Equal(t, "Test Address", createdOrder.ShippingAddressText)       // Сравниваем с исходным
	assert.True(t, now.Equal(createdOrder.CreatedAt), "CreatedAt mismatch") // Используем .Equal для time.Time
	assert.True(t, now.Equal(createdOrder.UpdatedAt), "UpdatedAt mismatch")

	require.Len(t, createdOrder.OrderItems, 1)
	item := createdOrder.OrderItems[0]
	assert.Equal(t, expectedOrderItemID1, item.ID)
	assert.Equal(t, expectedOrderID, item.OrderID)    // OrderID позиции должен быть равен ID заказа
	assert.Equal(t, productID1, item.ProductID)       // С исходным productID
	assert.Equal(t, 1, item.Quantity)                 // С исходным Quantity
	assert.InDelta(t, 10.0, item.PricePerUnit, 0.001) // С исходным PricePerUnit
	assert.True(t, now.Equal(item.CreatedAt), "OrderItem CreatedAt mismatch")
	assert.True(t, now.Equal(item.UpdatedAt), "OrderItem UpdatedAt mismatch")

	mockRepo.AssertExpectations(t)
}

func TestService_CreateOrder_Error_RepositoryFails(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	orderService := NewService(mockRepo)

	ctx := context.Background()

	userID := uuid.Must(uuid.NewV4())
	productID1 := uuid.Must(uuid.NewV4())
	orderInput := &Order{
		UserID: userID,
		OrderItems: []OrderItem{
			{ProductID: productID1, Quantity: 1, PricePerUnit: 10.0},
		},
		ShippingAddressText: "Test Address",
	}

	repoErr := errors.New("simulated repository error")

	mockRepo.On("CreateOrder", ctx, mock.MatchedBy(func(argOrder *Order) bool {
		return argOrder.ID == uuid.Nil &&
			argOrder.Status == StatusNew &&
			math.Abs(argOrder.TotalAmount-(1*10.0)) < 0.001 &&
			len(argOrder.OrderItems) == 1 &&
			argOrder.OrderItems[0].ID == uuid.Nil &&
			argOrder.OrderItems[0].OrderID == uuid.Nil
	})).Return(uuid.Nil, repoErr)

	createdOrder, err := orderService.CreateOrder(ctx, orderInput)

	require.Error(t, err)
	require.Nil(t, createdOrder)
	require.ErrorIs(t, err, repoErr)

	mockRepo.AssertExpectations(t)
}

// order-service/internal/order/service_test.go

// ... (предыдущие тесты) ...

func TestService_CreateOrder_Error_EmptyOrderItems(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	orderService := NewService(mockRepo)

	ctx := context.Background()

	userID := uuid.Must(uuid.NewV4())
	orderInput := &Order{
		UserID:              userID,
		OrderItems:          []OrderItem{},
		ShippingAddressText: "Test Address",
	}

	createdOrder, err := orderService.CreateOrder(ctx, orderInput)
	require.Error(t, err)
	require.Nil(t, createdOrder)
	assert.EqualError(t, err, "service: order must contain at least one item")
	mockRepo.AssertExpectations(t)
}

func TestService_CreateOrder_Error_InvalidOrderItem_ZeroQuantity(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	orderService := NewService(mockRepo)

	ctx := context.Background()

	userID := uuid.Must(uuid.NewV4())
	productID1 := uuid.Must(uuid.NewV4())
	orderInput := &Order{
		UserID: userID,
		OrderItems: []OrderItem{
			{ProductID: productID1, Quantity: 0, PricePerUnit: 10.0},
		},
		ShippingAddressText: "Test Address",
	}

	createdOrder, err := orderService.CreateOrder(ctx, orderInput)
	require.Error(t, err)
	require.Nil(t, createdOrder)
	assert.EqualError(t, err, fmt.Sprintf("service: order item quantity for product %s must be greater than zero", productID1))
	mockRepo.AssertExpectations(t)
}

func TestService_CreateOrder_Error_InvalidOrderItem_NegativePrice(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	orderService := NewService(mockRepo)

	ctx := context.Background()

	userID := uuid.Must(uuid.NewV4())
	productID1 := uuid.Must(uuid.NewV4())
	orderInput := &Order{
		UserID: userID,
		OrderItems: []OrderItem{
			{ProductID: productID1, Quantity: 1, PricePerUnit: -1.0},
		},
		ShippingAddressText: "Test Address",
	}

	createdOrder, err := orderService.CreateOrder(ctx, orderInput)
	require.Error(t, err)
	require.Nil(t, createdOrder)
	assert.EqualError(t, err, fmt.Sprintf("service: order item price per unit for product %s cannot be negative", productID1))
	mockRepo.AssertExpectations(t)
}

func TestService_CreateOrder_Error_InvalidOrderItem_NilProductID(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	orderService := NewService(mockRepo)

	ctx := context.Background()

	userID := uuid.Must(uuid.NewV4())
	orderInput := &Order{
		UserID: userID,
		OrderItems: []OrderItem{
			{ProductID: uuid.Nil, Quantity: 1, PricePerUnit: 10.0},
		},
		ShippingAddressText: "Test Address",
	}

	createdOrder, err := orderService.CreateOrder(ctx, orderInput)
	require.Error(t, err)
	require.Nil(t, createdOrder)
	assert.EqualError(t, err, "service: product id in order item cannot be nil")
	mockRepo.AssertExpectations(t)
}

func TestService_GetOrderByID_Success(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	orderService := NewService(mockRepo)
	ctx := context.Background()

	expectedOrderID := uuid.Must(uuid.NewV4())
	expectedUserID := uuid.Must(uuid.NewV4())
	expectedOrderItemID := uuid.Must(uuid.NewV4())
	expectedProductID := uuid.Must(uuid.NewV4())
	now := time.Now().UTC().Truncate(time.Microsecond)

	expectedOrder := &Order{
		ID:     expectedOrderID,
		UserID: expectedUserID,
		Status: StatusProcessing,
		OrderItems: []OrderItem{
			{
				ID:           expectedOrderItemID,
				OrderID:      expectedOrderID,
				ProductID:    expectedProductID,
				Quantity:     2,
				PricePerUnit: 10.0,
				CreatedAt:    now,
				UpdatedAt:    now,
			},
		},
		TotalAmount:         20.0,
		ShippingAddressText: "Shipping Address",
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	mockRepo.On("GetOrderByID", ctx, expectedOrderID).
		Return(expectedOrder, nil).
		Once()

	returnedOrder, err := orderService.GetOrderByID(ctx, expectedOrderID)
	require.NoError(t, err)
	require.NotNil(t, returnedOrder)
	assert.Equal(t, expectedOrderID, returnedOrder.ID)
	assert.Equal(t, expectedUserID, returnedOrder.UserID)
	assert.Equal(t, expectedOrder.Status, returnedOrder.Status)
	assert.Equal(t, expectedOrder.ShippingAddressText, returnedOrder.ShippingAddressText)
	assert.True(t, expectedOrder.CreatedAt.Equal(returnedOrder.CreatedAt))
	assert.True(t, expectedOrder.UpdatedAt.Equal(returnedOrder.UpdatedAt))

	assert.Len(t, expectedOrder.OrderItems, len(returnedOrder.OrderItems))
	for i := range expectedOrder.OrderItems {
		expectedItem := expectedOrder.OrderItems[i]
		returnedItem := returnedOrder.OrderItems[i] // Предполагаем, что порядок такой же (если мок вернул их в том же порядке)

		assert.Equal(t, expectedItem.ID, returnedItem.ID)
		assert.Equal(t, expectedItem.OrderID, returnedItem.OrderID)
		assert.Equal(t, expectedItem.ProductID, returnedItem.ProductID)
		assert.InDelta(t, expectedItem.PricePerUnit, returnedItem.PricePerUnit, 0.001)
		assert.Equal(t, expectedItem.Quantity, returnedItem.Quantity)
		assert.True(t, expectedItem.CreatedAt.Equal(returnedItem.CreatedAt))
		assert.True(t, expectedItem.UpdatedAt.Equal(returnedItem.UpdatedAt))
	}

	mockRepo.AssertExpectations(t)
}

func TestService_GetOrderByID_NotFound(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	orderService := NewService(mockRepo)
	ctx := context.Background()

	searchOrderID := uuid.Must(uuid.NewV4())

	mockRepo.On("GetOrderByID", ctx, searchOrderID).
		Return(nil, ErrOrderNotFound).
		Once()

	foundOrder, err := orderService.GetOrderByID(ctx, searchOrderID)

	require.Error(t, err, "Expected an error when order is not found")
	assert.Nil(t, foundOrder, "FoundOrder should be nil when order is not found")
	assert.ErrorIs(t, err, ErrOrderNotFound, "Error should be (or wrap) ErrOrderNotFound")

	mockRepo.AssertExpectations(t)
}

func TestService_GetOrderByID_RepoError(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	orderService := NewService(mockRepo)
	ctx := context.Background()

	searchOrderID := uuid.Must(uuid.NewV4())
	expectedRepoError := errors.New("simulated generic repository error")

	mockRepo.On("GetOrderByID", ctx, searchOrderID).
		Return(nil, expectedRepoError).
		Once()

	foundOrder, err := orderService.GetOrderByID(ctx, searchOrderID)

	require.Error(t, err, "Expected an error when repository fails")
	assert.Nil(t, foundOrder, "FoundOrder should be nil when repository fails")
	assert.ErrorIs(t, err, expectedRepoError, "Service error should wrap the original repository error")

	mockRepo.AssertExpectations(t)
}
func TestService_GetOrdersByUserID_Success(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	orderService := NewService(mockRepo)
	ctx := context.Background()

	userID := uuid.Must(uuid.NewV4())
	orderID1 := uuid.Must(uuid.NewV4())
	orderID2 := uuid.Must(uuid.NewV4())
	now := time.Now().UTC().Truncate(time.Microsecond)

	expectedOrders := []Order{
		{
			ID:     orderID1,
			UserID: userID,
			Status: StatusProcessing,
			OrderItems: []OrderItem{
				{
					ID:           uuid.Must(uuid.NewV4()),
					OrderID:      orderID1,
					ProductID:    uuid.Must(uuid.NewV4()),
					Quantity:     1,
					PricePerUnit: 20.0,
					CreatedAt:    now,
					UpdatedAt:    now,
				},
			},
			TotalAmount:         20.0,
			ShippingAddressText: "Shipping Address 1",
			CreatedAt:           now,
			UpdatedAt:           now,
		},
		{
			ID:     orderID2,
			UserID: userID,
			Status: StatusProcessing,
			OrderItems: []OrderItem{
				{
					ID:           uuid.Must(uuid.NewV4()),
					OrderID:      orderID2,
					ProductID:    uuid.Must(uuid.NewV4()),
					Quantity:     2,
					PricePerUnit: 15.0,
					CreatedAt:    now,
					UpdatedAt:    now,
				},
			},
			TotalAmount:         30.0,
			ShippingAddressText: "Shipping Address 2",
			CreatedAt:           now,
			UpdatedAt:           now,
		},
	}

	mockRepo.On("GetOrdersByUserID", ctx, userID).
		Return(expectedOrders, nil).
		Once()

	returnedOrders, err := orderService.GetOrdersByUserID(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, returnedOrders)
	require.Len(t, expectedOrders, len(returnedOrders))

	for i := range expectedOrders {
		expectedOrder := expectedOrders[i]
		returnedOrder := returnedOrders[i]

		assert.Equal(t, expectedOrder.ID, returnedOrder.ID)
		assert.Equal(t, expectedOrder.UserID, returnedOrder.UserID)
		assert.Equal(t, expectedOrder.Status, returnedOrder.Status)
		assert.Equal(t, expectedOrder.ShippingAddressText, returnedOrder.ShippingAddressText)
		assert.InDelta(t, expectedOrder.TotalAmount, returnedOrder.TotalAmount, 0.001)
		assert.True(t, expectedOrder.CreatedAt.Equal(returnedOrder.CreatedAt))
		assert.True(t, expectedOrder.UpdatedAt.Equal(returnedOrder.UpdatedAt))

		assert.Len(t, expectedOrder.OrderItems, len(returnedOrder.OrderItems))
		for j := range expectedOrder.OrderItems {
			expectedItem := expectedOrder.OrderItems[j]
			returnedItem := returnedOrder.OrderItems[j]

			assert.Equal(t, expectedItem.ID, returnedItem.ID)
			assert.Equal(t, expectedItem.OrderID, returnedItem.OrderID)
			assert.Equal(t, expectedItem.ProductID, returnedItem.ProductID)
			assert.Equal(t, expectedItem.Quantity, returnedItem.Quantity)
			assert.InDelta(t, expectedItem.PricePerUnit, returnedItem.PricePerUnit, 0.001)
			assert.True(t, expectedItem.CreatedAt.Equal(returnedItem.UpdatedAt))
			assert.True(t, expectedItem.UpdatedAt.Equal(returnedItem.UpdatedAt))
		}
	}

	mockRepo.AssertExpectations(t)
}

func TestService_GetOrdersByUserID_RepoError(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	orderService := NewService(mockRepo)
	ctx := context.Background()

	userID := uuid.Must(uuid.NewV4())
	expectedRepoError := errors.New("simulated repository error for GetOrdersByUserID")

	mockRepo.On("GetOrdersByUserID", ctx, userID).
		Return(nil, expectedRepoError).
		Once()

	foundOrders, err := orderService.GetOrdersByUserID(ctx, userID)

	require.Error(t, err, "Expected an error when repository fails")
	assert.Nil(t, foundOrders, "FoundOrders should be nil when repository fails")

	assert.ErrorIs(t, err, expectedRepoError, "Service error should wrap the original repository error")
	assert.Contains(t, err.Error(), "service: failed to fetch user orders", "Error message mismatch")

	mockRepo.AssertExpectations(t)
}

func TestService_UpdateOrderStatus_Success(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	orderService := NewService(mockRepo)
	ctx := context.Background()

	orderID := uuid.Must(uuid.NewV4())
	currentStatus := StatusNew
	newStatus := StatusProcessing

	mockCurrentOrder := &Order{
		ID:         orderID,
		UserID:     uuid.Must(uuid.NewV4()),
		Status:     currentStatus,
		OrderItems: []OrderItem{},
	}

	mockRepo.On("GetOrderByID", ctx, orderID).
		Return(mockCurrentOrder, nil).
		Once()

	mockRepo.On("UpdateOrderStatus", ctx, orderID, newStatus).
		Return(nil).
		Once()

	err := orderService.UpdateOrderStatus(ctx, orderID, newStatus)
	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_UpdateOrderStatus_OrderNotFound(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	orderService := NewService(mockRepo)
	ctx := context.Background()

	nonExistingOrderID := uuid.Must(uuid.NewV4())
	newStatus := StatusProcessing

	mockRepo.On("GetOrderByID", ctx, nonExistingOrderID).
		Return(nil, ErrOrderNotFound).
		Once()

	err := orderService.UpdateOrderStatus(ctx, nonExistingOrderID, newStatus)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrOrderNotFound)
	mockRepo.AssertExpectations(t)
}

func TestService_UpdateOrderStatus_StatusAlreadySet(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	orderService := NewService(mockRepo)
	ctx := context.Background()

	orderID := uuid.Must(uuid.NewV4())
	currentStatus := StatusProcessing
	newStatus := StatusProcessing

	mockCurrentOrder := &Order{
		ID:         orderID,
		UserID:     uuid.Must(uuid.NewV4()),
		Status:     currentStatus,
		OrderItems: []OrderItem{},
	}

	mockRepo.On("GetOrderByID", ctx, orderID).
		Return(mockCurrentOrder, nil).
		Once()

	err := orderService.UpdateOrderStatus(ctx, orderID, newStatus)
	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_UpdateOrderStatus_InvalidTransition(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	orderService := NewService(mockRepo)
	ctx := context.Background()

	orderID := uuid.Must(uuid.NewV4())
	currentStatus := StatusProcessing
	newStatus := StatusNew

	mockCurrentOrder := &Order{
		ID:         orderID,
		UserID:     uuid.Must(uuid.NewV4()),
		Status:     currentStatus,
		OrderItems: []OrderItem{},
	}

	mockRepo.On("GetOrderByID", ctx, orderID).
		Return(mockCurrentOrder, nil).
		Once()

	err := orderService.UpdateOrderStatus(ctx, orderID, newStatus)
	assert.Error(t, err)
	assert.EqualError(t, err, fmt.Sprintf("service: invalid status transition from %s to %s (or no rules for %s)", currentStatus, newStatus, currentStatus))
	mockRepo.AssertExpectations(t)
}

func TestService_UpdateOrderStatus_RepoUpdateError(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	orderService := NewService(mockRepo)
	ctx := context.Background()

	orderID := uuid.Must(uuid.NewV4())
	newStatus := StatusProcessing

	mockErr := errors.New("simulate repo error")

	mockRepo.On("GetOrderByID", ctx, orderID).
		Return(nil, mockErr).
		Once()

	err := orderService.UpdateOrderStatus(ctx, orderID, newStatus)
	require.Error(t, err)
	require.ErrorIs(t, err, mockErr)
	mockRepo.AssertExpectations(t)
}

func TestService_UpdateOrderStatus_RepoFailsOnUpdateCall(t *testing.T) {
	mockRepo := new(MockOrderRepository)
	orderService := NewService(mockRepo)
	ctx := context.Background()

	orderID := uuid.Must(uuid.NewV4())
	currentStatus := StatusNew
	newStatus := StatusProcessing

	mockCurrentOrder := &Order{
		ID:         orderID,
		UserID:     uuid.Must(uuid.NewV4()),
		Status:     currentStatus,
		OrderItems: []OrderItem{},
	}

	mockRepo.On("GetOrderByID", ctx, orderID).
		Return(mockCurrentOrder, nil).
		Once()

	repoUpdateErr := errors.New("repo failed during actual update")

	mockRepo.On("UpdateOrderStatus", ctx, orderID, newStatus).
		Return(repoUpdateErr).
		Once()

	err := orderService.UpdateOrderStatus(ctx, orderID, newStatus)
	require.Error(t, err)
	require.ErrorIs(t, err, repoUpdateErr)
	mockRepo.AssertExpectations(t)
}
