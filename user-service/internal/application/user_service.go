package application

import (
	"errors"
	"fmt"
	"strings"
	"user-service/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

type UserRepository interface {
	Create(user *domain.User) error
	GetByEmail(email string) (*domain.User, error)
}

type UserService struct {
	repo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Register(user *domain.User) error {
	user.Password = strings.TrimSpace(user.Password)
	if user.Password == "" {
		return errors.New("password required")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)

	return s.repo.Create(user)
}

func (s *UserService) Login(email, password string) (*domain.User, error) {

	user, err := s.repo.GetByEmail(email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		fmt.Printf("Bcrypt compare error: %v\n", err)
		return nil, errors.New("invalid credentials")
	}

	return user, nil
}
