package services

import (
	"errors"

	"auto-store-api/internal/models"
	"auto-store-api/internal/repositories"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	userRepo    *repositories.UserRepository
	addressRepo *repositories.AddressRepository
	db          *gorm.DB
}

func NewUserService(userRepo *repositories.UserRepository, addressRepo *repositories.AddressRepository, db *gorm.DB) *UserService {
	return &UserService{userRepo: userRepo, addressRepo: addressRepo, db: db}
}

func (s *UserService) GetByID(id uuid.UUID) (*models.User, error) {
	return s.userRepo.GetByID(id)
}

func (s *UserService) UpdateProfile(user *models.User, firstName, lastName, phone *string) error {
	if firstName != nil {
		user.FirstName = *firstName
	}
	if lastName != nil {
		user.LastName = *lastName
	}
	if phone != nil {
		user.Phone = *phone
	}
	return s.userRepo.Update(user)
}

func (s *UserService) ListAddresses(userID uuid.UUID) ([]models.Address, error) {
	return s.addressRepo.GetByUserID(userID)
}

func (s *UserService) AddAddress(userID uuid.UUID, addr *models.Address) error {
	addr.UserID = userID
	return s.addressRepo.Create(addr)
}

func (s *UserService) UpdateAddress(addr *models.Address) error {
	return s.addressRepo.Update(addr)
}

func (s *UserService) DeleteAddress(id, userID uuid.UUID) error {
	return s.addressRepo.Delete(id, userID)
}

func (s *UserService) GetAddress(id, userID uuid.UUID) (*models.Address, error) {
	return s.addressRepo.GetByIDAndUser(id, userID)
}

// UpdateRole sets a user's role. Only valid roles are ADMIN, VENDOR, CUSTOMER (stored in caps).
func (s *UserService) UpdateRole(userID uuid.UUID, role models.Role) error {
	if role != models.RoleAdmin && role != models.RoleVendor && role != models.RoleCustomer {
		return errors.New("invalid role: must be ADMIN, VENDOR, or CUSTOMER")
	}
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return err
	}
	user.Role = role
	return s.userRepo.Update(user)
}

func (s *UserService) ChangePassword(userID uuid.UUID, currentPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.PasswordHash = string(hash)
	return s.userRepo.Update(user)
}
