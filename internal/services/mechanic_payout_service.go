package services

import (
	"auto-store-api/internal/config"
	"auto-store-api/internal/models"
	"auto-store-api/internal/repositories"
	"auto-store-api/pkg/paystack"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrMechanicPayoutNotConfigured = errors.New("mechanic payout account is not configured")
	ErrMechanicPayoutInvalidBank   = errors.New("invalid bank code or account number")
)

// MechanicPayoutStatus describes Paystack subaccount setup for a mechanic.
type MechanicPayoutStatus struct {
	Configured         bool       `json:"configured"`
	SubaccountCode     string     `json:"subaccount_code,omitempty"`
	AccountName        string     `json:"account_name,omitempty"`
	AccountNumberLast4 string     `json:"account_number_last4,omitempty"`
	BankCode           string     `json:"bank_code,omitempty"`
	ConfiguredAt       *time.Time `json:"configured_at,omitempty"`
}

type MechanicPayoutService struct {
	cfg          config.PaystackConfig
	client       *paystack.Client
	mechanicRepo *repositories.MechanicRepository
	userRepo     *repositories.UserRepository
	log          *zap.Logger
}

func NewMechanicPayoutService(
	cfg config.PaystackConfig,
	mechanicRepo *repositories.MechanicRepository,
	userRepo *repositories.UserRepository,
	log *zap.Logger,
) *MechanicPayoutService {
	var client *paystack.Client
	if cfg.Enabled {
		client = paystack.NewClient(cfg.SecretKey)
	}
	return &MechanicPayoutService{
		cfg:          cfg,
		client:       client,
		mechanicRepo: mechanicRepo,
		userRepo:     userRepo,
		log:          log,
	}
}

func (s *MechanicPayoutService) requireClient() error {
	if s.client == nil || !s.cfg.Enabled {
		return ErrPaystackNotConfigured
	}
	return nil
}

func (s *MechanicPayoutService) ListBanks() ([]paystack.Bank, error) {
	if err := s.requireClient(); err != nil {
		return nil, err
	}
	resp, err := s.client.ListBanks(s.cfg.BankCountry)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (s *MechanicPayoutService) GetStatus(userID uuid.UUID) (*MechanicPayoutStatus, error) {
	profile, err := s.mechanicRepo.GetProfileByUserID(userID)
	if err != nil {
		return nil, ErrMechanicProfileNotFound
	}
	return profileToPayoutStatus(profile), nil
}

func (s *MechanicPayoutService) Setup(userID uuid.UUID, bankCode, accountNumber string) (*MechanicPayoutStatus, error) {
	if err := s.requireClient(); err != nil {
		return nil, err
	}
	bankCode = strings.TrimSpace(bankCode)
	accountNumber = strings.TrimSpace(accountNumber)
	if bankCode == "" || accountNumber == "" {
		return nil, ErrMechanicPayoutInvalidBank
	}

	profile, err := s.mechanicRepo.GetProfileByUserID(userID)
	if err != nil {
		return nil, ErrMechanicProfileNotFound
	}
	if !profile.IsVerified() {
		return nil, errors.New("mechanic profile must be verified before configuring payout")
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil || user == nil {
		return nil, errors.New("user not found")
	}

	contactName := strings.TrimSpace(user.FirstName + " " + user.LastName)
	if contactName == "" {
		contactName = profile.BusinessName
	}

	var resp *paystack.SubaccountResponse
	if profile.PaystackSubaccountCode != "" {
		resp, err = s.client.UpdateSubaccount(profile.PaystackSubaccountCode, bankCode, accountNumber)
	} else {
		resp, err = s.client.CreateSubaccount(paystack.CreateSubaccountParams{
			BusinessName:        profile.BusinessName,
			BankCode:            bankCode,
			AccountNumber:       accountNumber,
			PercentageCharge:    0, // per-txn split via transaction_charge on initialize
			PrimaryContactName:  contactName,
			PrimaryContactEmail: user.Email,
		})
	}
	if err != nil {
		s.log.Warn("paystack subaccount setup failed", zap.Error(err), zap.String("mechanic_profile_id", profile.ID.String()))
		return nil, err
	}

	now := time.Now()
	profile.PaystackSubaccountCode = resp.Data.SubaccountCode
	profile.PaystackBankCode = bankCode
	profile.PaystackAccountNumber = accountNumber
	profile.PaystackAccountName = resp.Data.AccountName
	profile.PayoutConfiguredAt = &now

	if err := s.mechanicRepo.UpdateProfile(profile); err != nil {
		return nil, err
	}
	return profileToPayoutStatus(profile), nil
}

func profileToPayoutStatus(p *models.MechanicProfile) *MechanicPayoutStatus {
	st := &MechanicPayoutStatus{
		Configured:     p.PaystackSubaccountCode != "",
		SubaccountCode: p.PaystackSubaccountCode,
		AccountName:    p.PaystackAccountName,
		BankCode:       p.PaystackBankCode,
		ConfiguredAt:   p.PayoutConfiguredAt,
	}
	if len(p.PaystackAccountNumber) >= 4 {
		st.AccountNumberLast4 = p.PaystackAccountNumber[len(p.PaystackAccountNumber)-4:]
	}
	return st
}
