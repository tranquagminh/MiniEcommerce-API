package application

import (
	"context"
	"errors"
	"testing"
	"user-service/internal/domain"

	"gorm.io/gorm"
)

// Mock implementations
type MockUserRepository struct {
	users       map[uint]*domain.User
	emailIndex  map[string]*domain.User
	nextID      uint
	shouldError bool
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users:      make(map[uint]*domain.User),
		emailIndex: make(map[string]*domain.User),
		nextID:     1,
	}
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	if m.shouldError {
		return errors.New("mock error")
	}
	user.ID = m.nextID
	m.nextID++
	m.users[user.ID] = user
	m.emailIndex[user.Email] = user
	return nil
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.shouldError {
		return nil, errors.New("mock error")
	}
	user, ok := m.emailIndex[email]
	if !ok {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uint) (*domain.User, error) {
	if m.shouldError {
		return nil, errors.New("mock error")
	}
	user, ok := m.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (m *MockUserRepository) Update(ctx context.Context, user *domain.User) error {
	if m.shouldError {
		return errors.New("mock error")
	}
	if _, ok := m.users[user.ID]; !ok {
		return errors.New("user not found")
	}
	m.users[user.ID] = user
	m.emailIndex[user.Email] = user
	return nil
}

func (m *MockUserRepository) UpdateFields(ctx context.Context, id uint, fields map[string]interface{}) error {
	if m.shouldError {
		return errors.New("mock error")
	}
	user, ok := m.users[id]
	if !ok {
		return errors.New("user not found")
	}
	// Simple mock - just update the user
	m.users[id] = user
	return nil
}

func (m *MockUserRepository) SoftDelete(ctx context.Context, id uint) error {
	if m.shouldError {
		return errors.New("mock error")
	}
	if _, ok := m.users[id]; !ok {
		return errors.New("user not found")
	}
	delete(m.users, id)
	return nil
}

func (m *MockUserRepository) ExistsEmail(ctx context.Context, email string) (bool, error) {
	if m.shouldError {
		return false, errors.New("mock error")
	}
	_, exists := m.emailIndex[email]
	return exists, nil
}

func (m *MockUserRepository) List(ctx context.Context, offset, limit int) ([]*domain.User, int64, error) {
	if m.shouldError {
		return nil, 0, errors.New("mock error")
	}
	users := make([]*domain.User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, user)
	}
	return users, int64(len(users)), nil
}

func (m *MockUserRepository) WithTx(tx *gorm.DB) UserRepository {
	return m
}

type MockTransactionManager struct {
	shouldError bool
}

func NewMockTransactionManager() *MockTransactionManager {
	return &MockTransactionManager{}
}

func (m *MockTransactionManager) ExecuteInTx(ctx context.Context, fn func(tx *gorm.DB) error) error {
	if m.shouldError {
		return errors.New("transaction error")
	}
	return fn(nil)
}

type MockPasswordValidator struct {
	shouldError bool
}

func (m *MockPasswordValidator) Validate(password string) error {
	if m.shouldError {
		return errors.New("weak password")
	}
	return nil
}

// Tests
func TestUserService_Register(t *testing.T) {
	tests := []struct {
		name      string
		user      *domain.User
		wantErr   bool
		setupMock func(*MockUserRepository, *MockPasswordValidator)
	}{
		{
			name: "successful registration",
			user: &domain.User{
				Email:    "test@example.com",
				Username: "testuser",
				Password: "StrongP@ss123",
			},
			wantErr: false,
			setupMock: func(repo *MockUserRepository, validator *MockPasswordValidator) {
				// No setup needed
			},
		},
		{
			name: "duplicate email",
			user: &domain.User{
				Email:    "existing@example.com",
				Username: "testuser",
				Password: "StrongP@ss123",
			},
			wantErr: true,
			setupMock: func(repo *MockUserRepository, validator *MockPasswordValidator) {
				// Pre-create user with same email
				repo.emailIndex["existing@example.com"] = &domain.User{
					ID:    1,
					Email: "existing@example.com",
				}
			},
		},
		{
			name: "weak password",
			user: &domain.User{
				Email:    "test@example.com",
				Username: "testuser",
				Password: "weak",
			},
			wantErr: true,
			setupMock: func(repo *MockUserRepository, validator *MockPasswordValidator) {
				validator.shouldError = true
			},
		},
		{
			name: "empty password",
			user: &domain.User{
				Email:    "test@example.com",
				Username: "testuser",
				Password: "",
			},
			wantErr: true,
			setupMock: func(repo *MockUserRepository, validator *MockPasswordValidator) {
				// No setup needed
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockUserRepository()
			txManager := NewMockTransactionManager()
			validator := &MockPasswordValidator{}

			if tt.setupMock != nil {
				tt.setupMock(repo, validator)
			}

			service := NewUserService(repo, txManager, nil, validator)
			err := service.Register(context.Background(), tt.user)

			if (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && tt.user.ID == 0 {
				t.Errorf("Register() should set user ID")
			}
		})
	}
}

func TestUserService_Login(t *testing.T) {
	repo := NewMockUserRepository()
	txManager := NewMockTransactionManager()
	validator := &MockPasswordValidator{}
	service := NewUserService(repo, txManager, nil, validator)

	// Create a test user with hashed password
	testUser := &domain.User{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "StrongP@ss123",
	}
	_ = service.Register(context.Background(), testUser)

	tests := []struct {
		name     string
		email    string
		password string
		wantErr  bool
	}{
		{
			name:     "successful login",
			email:    "test@example.com",
			password: "StrongP@ss123",
			wantErr:  false,
		},
		{
			name:     "wrong password",
			email:    "test@example.com",
			password: "WrongPassword",
			wantErr:  true,
		},
		{
			name:     "non-existent user",
			email:    "notfound@example.com",
			password: "SomePassword",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := service.Login(context.Background(), tt.email, tt.password)

			if (err != nil) != tt.wantErr {
				t.Errorf("Login() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && user == nil {
				t.Errorf("Login() should return user on success")
			}
		})
	}
}

func TestUserService_GetUser(t *testing.T) {
	repo := NewMockUserRepository()
	txManager := NewMockTransactionManager()
	validator := &MockPasswordValidator{}
	service := NewUserService(repo, txManager, nil, validator)

	// Create a test user
	testUser := &domain.User{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "StrongP@ss123",
	}
	_ = service.Register(context.Background(), testUser)

	t.Run("get existing user", func(t *testing.T) {
		user, err := service.GetUser(context.Background(), testUser.ID)
		if err != nil {
			t.Errorf("GetUser() error = %v", err)
		}
		if user == nil {
			t.Errorf("GetUser() should return user")
			return
		}
		if user.Email != testUser.Email {
			t.Errorf("GetUser() email = %v, want %v", user.Email, testUser.Email)
		}
	})

	t.Run("get non-existent user", func(t *testing.T) {
		_, err := service.GetUser(context.Background(), 9999)
		if err == nil {
			t.Errorf("GetUser() should return error for non-existent user")
		}
	})
}

func TestUserService_UpdateUser(t *testing.T) {
	repo := NewMockUserRepository()
	txManager := NewMockTransactionManager()
	validator := &MockPasswordValidator{}
	service := NewUserService(repo, txManager, nil, validator)

	// Create a test user
	testUser := &domain.User{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "StrongP@ss123",
	}
	_ = service.Register(context.Background(), testUser)

	t.Run("update existing user", func(t *testing.T) {
		testUser.FirstName = "John"
		testUser.LastName = "Doe"

		err := service.UpdateUser(context.Background(), testUser)
		if err != nil {
			t.Errorf("UpdateUser() error = %v", err)
		}

		// Verify update
		updated, _ := service.GetUser(context.Background(), testUser.ID)
		if updated.FirstName != "John" {
			t.Errorf("UpdateUser() FirstName = %v, want %v", updated.FirstName, "John")
		}
	})
}

func TestUserService_DeleteUser(t *testing.T) {
	repo := NewMockUserRepository()
	txManager := NewMockTransactionManager()
	validator := &MockPasswordValidator{}
	service := NewUserService(repo, txManager, nil, validator)

	// Create a test user
	testUser := &domain.User{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "StrongP@ss123",
	}
	_ = service.Register(context.Background(), testUser)

	t.Run("delete existing user", func(t *testing.T) {
		err := service.DeleteUser(context.Background(), testUser.ID)
		if err != nil {
			t.Errorf("DeleteUser() error = %v", err)
		}

		// Verify deletion
		_, err = service.GetUser(context.Background(), testUser.ID)
		if err == nil {
			t.Errorf("GetUser() should return error for deleted user")
		}
	})
}

func TestUserService_ListUsers(t *testing.T) {
	repo := NewMockUserRepository()
	txManager := NewMockTransactionManager()
	validator := &MockPasswordValidator{}
	service := NewUserService(repo, txManager, nil, validator)

	// Create test users
	for i := 1; i <= 5; i++ {
		user := &domain.User{
			Email:    "user" + string(rune(i)) + "@example.com",
			Username: "user" + string(rune(i)),
			Password: "StrongP@ss123",
		}
		_ = service.Register(context.Background(), user)
	}

	t.Run("list users", func(t *testing.T) {
		users, total, err := service.ListUsers(context.Background(), 1, 10)
		if err != nil {
			t.Errorf("ListUsers() error = %v", err)
		}
		if len(users) != 5 {
			t.Errorf("ListUsers() returned %d users, want 5", len(users))
		}
		if total != 5 {
			t.Errorf("ListUsers() total = %d, want 5", total)
		}
	})
}
