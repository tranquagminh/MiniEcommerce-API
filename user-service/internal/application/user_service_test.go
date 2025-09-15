package application

import (
	"errors"
	"testing"
	"user-service/internal/domain"

	"golang.org/x/crypto/bcrypt"
	"github.com/stretchr/testify/assert"

)

// Mock UserRepository để test
type mockUserRepo struct {
	users map[string]*domain.User
}

func newMockRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*domain.User)}
}

func (m *mockUserRepo) Create(user *domain.User) error {
	// giả lập insert vào "db"
	if _, exists := m.users[user.Email]; exists {
		return errors.New("duplicate email")
	}
	m.users[user.Email] = user
	return nil
}

func (m *mockUserRepo) GetByEmail(email string) (*domain.User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}


func TestUserService_Register(t *testing.T) {
	repo := newMockRepo()
	service := NewUserService(repo)

	user := &domain.User{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "secret123",
	}

	err := service.Register(user)
	assert.NoError(t, err)

	// đảm bảo user được lưu vào repo
	saved, _ := repo.GetByEmail("alice@example.com")
	assert.NotNil(t, saved)

	// đảm bảo password được hash (không lưu plaintext)
	assert.NotEqual(t, "secret123", saved.Password)

	// validate hash có thể match password
	err = bcrypt.CompareHashAndPassword([]byte(saved.Password), []byte("secret123"))
	assert.NoError(t, err)
}

func TestUserService_Login_Success(t *testing.T) {
	repo := newMockRepo()
	service := NewUserService(repo)

	// Đăng ký user
	user := &domain.User{
		Username: "bob",
		Email:    "bob@example.com",
		Password: "mypassword",
	}
	_ = service.Register(user)

	// Thử login với password đúng
	loggedIn, err := service.Login("bob@example.com", "mypassword")
	assert.NoError(t, err)
	assert.Equal(t, "bob", loggedIn.Username)
}

func TestUserService_Login_WrongPassword(t *testing.T) {
	repo := newMockRepo()
	service := NewUserService(repo)

	// Đăng ký user
	user := &domain.User{
		Username: "bob",
		Email:    "bob@example.com",
		Password: "mypassword",
	}
	_ = service.Register(user)

	// Login với password sai
	_, err := service.Login("bob@example.com", "wrongpass")
	assert.Error(t, err)
}

func TestUserService_Login_UserNotFound(t *testing.T) {
	repo := newMockRepo()
	service := NewUserService(repo)

	_, err := service.Login("idontexist@example.com", "pass")
	assert.Error(t, err)
}