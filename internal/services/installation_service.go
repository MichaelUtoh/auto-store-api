package services

import (
	"context"
	"errors"
	"time"

	"auto-store-api/internal/models"
	"auto-store-api/internal/repositories"
	"auto-store-api/pkg/geo"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrInstallationJobTypeNotFound = errors.New("installation job type not found")
	ErrInstallationQuoteNotFound   = errors.New("installation quote not found")
	ErrInstallationQuoteExpired    = errors.New("installation quote has expired")
	ErrInstallationQuoteNotReady   = errors.New("installation quote is not available for booking")
	ErrInstallationQuoteLineNotFound = errors.New("installation quote line not found")
	ErrNoMechanicsInArea           = errors.New("no verified mechanics available in your area for this job")
	ErrInstallationBookingNotFound = errors.New("installation booking not found")
	ErrBookingNotCancellable         = errors.New("booking cannot be cancelled in its current status")
	ErrInvalidBookingStatusTransition = errors.New("invalid booking status transition")
	ErrNoInstallableProducts         = errors.New("no installation-eligible products found")
	ErrOrderNotFoundForInstall       = errors.New("order not found")
	ErrOrderNotOwned                 = errors.New("order does not belong to user")
	ErrQuoteLineNotForMechanic       = errors.New("quote line does not belong to this mechanic")
	ErrScheduledInPast               = errors.New("scheduled time must be in the future")
)

const (
	quoteExpiryHours   = 24
	platformFeePercent = 0.10
	defaultSearchRadiusKm = 50.0
)

type CreateQuoteInput struct {
	OrderID           *uuid.UUID
	ProductIDs        []uuid.UUID
	VehicleMake       string
	VehicleModel      string
	VehicleYear       int
	ServiceStreet     string
	ServiceCity       string
	ServiceState      string
	ServicePostalCode string
	ServiceCountry    string
	Latitude          *float64
	Longitude         *float64
	Notes             string
	SearchRadiusKm    float64
}

type RespondQuoteLineInput struct {
	LaborPrice      *float64
	EstimatedHours  *float64
	MechanicMessage *string
}

type CreateBookingInput struct {
	QuoteID     uuid.UUID
	QuoteLineID uuid.UUID
	ScheduledAt time.Time
}

type InstallationService struct {
	installRepo *repositories.InstallationRepository
	orderRepo   *repositories.OrderRepository
	productRepo *repositories.ProductRepository
	mechanicRepo *repositories.MechanicRepository
	notifier    *Notifier
	log         *zap.Logger
	db          *gorm.DB
}

func NewInstallationService(
	installRepo *repositories.InstallationRepository,
	orderRepo *repositories.OrderRepository,
	productRepo *repositories.ProductRepository,
	mechanicRepo *repositories.MechanicRepository,
	notifier *Notifier,
	log *zap.Logger,
	db *gorm.DB,
) *InstallationService {
	return &InstallationService{
		installRepo: installRepo,
		orderRepo:   orderRepo,
		productRepo: productRepo,
		mechanicRepo: mechanicRepo,
		notifier:    notifier,
		log:         log,
		db:          db,
	}
}

func (s *InstallationService) ListJobTypes() ([]models.InstallationJobType, error) {
	return s.installRepo.ListJobTypes(true)
}

func (s *InstallationService) CreateQuote(userID uuid.UUID, input CreateQuoteInput) (*models.InstallationQuote, error) {
	if input.Latitude == nil || input.Longitude == nil {
		return nil, errors.New("latitude and longitude are required for mechanic matching")
	}
	_ = s.installRepo.ExpireQuotesBefore(time.Now())

	items, partsTotal, err := s.resolveQuoteItems(userID, input)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrNoInstallableProducts
	}

	radius := input.SearchRadiusKm
	if radius <= 0 {
		radius = defaultSearchRadiusKm
	}

	jobTypeIDs := uniqueJobTypeIDs(items)
	lines, err := s.buildQuoteLines(*input.Latitude, *input.Longitude, radius, jobTypeIDs)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, ErrNoMechanicsInArea
	}

	expiresAt := time.Now().Add(quoteExpiryHours * time.Hour)
	quote := &models.InstallationQuote{
		UserID:            userID,
		OrderID:           input.OrderID,
		Status:            models.QuoteStatusReady,
		VehicleMake:       input.VehicleMake,
		VehicleModel:      input.VehicleModel,
		VehicleYear:       input.VehicleYear,
		ServiceStreet:     input.ServiceStreet,
		ServiceCity:       input.ServiceCity,
		ServiceState:      input.ServiceState,
		ServicePostalCode: input.ServicePostalCode,
		ServiceCountry:    input.ServiceCountry,
		Latitude:          input.Latitude,
		Longitude:         input.Longitude,
		Notes:             input.Notes,
		ExpiresAt:         expiresAt,
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(quote).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].QuoteID = quote.ID
		}
		if err := tx.Create(&items).Error; err != nil {
			return err
		}
		for i := range lines {
			lines[i].QuoteID = quote.ID
		}
		if err := tx.Create(&lines).Error; err != nil {
			return err
		}
		_ = partsTotal
		return nil
	})
	if err != nil {
		return nil, err
	}

	loaded, err := s.installRepo.GetQuoteByID(quote.ID)
	if err != nil {
		return nil, err
	}
	s.emitQuoteReady(userID, quote.ID)
	return loaded, nil
}

func (s *InstallationService) resolveQuoteItems(userID uuid.UUID, input CreateQuoteInput) ([]models.InstallationQuoteItem, float64, error) {
	var productIDs []uuid.UUID
	var partsTotal float64

	if input.OrderID != nil {
		order, err := s.orderRepo.GetByID(*input.OrderID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, 0, ErrOrderNotFoundForInstall
			}
			return nil, 0, err
		}
		if order.UserID != userID {
			return nil, 0, ErrOrderNotOwned
		}
		for _, oi := range order.OrderItems {
			productIDs = append(productIDs, oi.ProductID)
			partsTotal += oi.TotalPrice
		}
	} else {
		productIDs = input.ProductIDs
	}

	var items []models.InstallationQuoteItem
	for _, pid := range productIDs {
		product, err := s.productRepo.GetByID(pid)
		if err != nil || product == nil {
			continue
		}
		if !product.InstallationEligible || product.InstallationJobTypeID == nil {
			continue
		}
		items = append(items, models.InstallationQuoteItem{
			ProductID: pid,
			JobTypeID: *product.InstallationJobTypeID,
			Quantity:  1,
		})
		if input.OrderID == nil {
			partsTotal += product.Price
		}
	}
	return items, partsTotal, nil
}

func (s *InstallationService) buildQuoteLines(lat, lng, radiusKm float64, jobTypeIDs []uuid.UUID) ([]models.InstallationQuoteLine, error) {
	mechanics, err := s.installRepo.ListVerifiedMechanicsWithCoords()
	if err != nil {
		return nil, err
	}

	jobTypes, err := s.installRepo.ListJobTypes(true)
	if err != nil {
		return nil, err
	}
	jobTypeMap := make(map[uuid.UUID]models.InstallationJobType, len(jobTypes))
	for _, jt := range jobTypes {
		jobTypeMap[jt.ID] = jt
	}

	needed := make(map[uuid.UUID]bool, len(jobTypeIDs))
	for _, id := range jobTypeIDs {
		needed[id] = true
	}

	var lines []models.InstallationQuoteLine
	seen := make(map[string]bool)

	for _, m := range mechanics {
		if m.Latitude == nil || m.Longitude == nil {
			continue
		}
		dist := geo.HaversineKm(lat, lng, *m.Latitude, *m.Longitude)
		effectiveRadius := m.ServiceRadiusKm
		if effectiveRadius <= 0 {
			effectiveRadius = 25
		}
		if dist > effectiveRadius || dist > radiusKm {
			continue
		}

		services, err := s.installRepo.ListInstallServicesForMechanic(m.ID)
		if err != nil {
			continue
		}
		serviceByJob := make(map[uuid.UUID]models.MechanicInstallService, len(services))
		for _, svc := range services {
			serviceByJob[svc.JobTypeID] = svc
		}

		for jtID := range needed {
			if len(services) > 0 {
				svc, ok := serviceByJob[jtID]
				if !ok || !svc.IsActive {
					continue
				}
			}
			key := m.ID.String() + ":" + jtID.String()
			if seen[key] {
				continue
			}
			seen[key] = true

			jt, ok := jobTypeMap[jtID]
			if !ok {
				continue
			}
			laborPrice := jt.BaseLaborPrice
			estimatedHours := float64(jt.BaseLaborMinutes) / 60.0
			if svc, ok := serviceByJob[jtID]; ok && svc.LaborPrice > 0 {
				laborPrice = svc.LaborPrice
			}

			lines = append(lines, models.InstallationQuoteLine{
				MechanicProfileID: m.ID,
				JobTypeID:         jtID,
				LaborPrice:        laborPrice,
				EstimatedHours:    estimatedHours,
				MechanicMessage:   "Estimate based on standard labor for this job type.",
				DistanceKm:        dist,
				Status:            models.QuoteLineStatusOffered,
			})
		}
	}
	return lines, nil
}

func (s *InstallationService) GetQuoteForUser(userID, quoteID uuid.UUID) (*models.InstallationQuote, error) {
	_ = s.installRepo.ExpireQuotesBefore(time.Now())
	quote, err := s.installRepo.GetQuoteByIDForUser(quoteID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInstallationQuoteNotFound
		}
		return nil, err
	}
	if quote.Status == models.QuoteStatusExpired || time.Now().After(quote.ExpiresAt) {
		return nil, ErrInstallationQuoteExpired
	}
	return quote, nil
}

func (s *InstallationService) ListQuotesForUser(userID uuid.UUID, page, limit int) ([]models.InstallationQuote, int64, error) {
	_ = s.installRepo.ExpireQuotesBefore(time.Now())
	return s.installRepo.ListQuotesForUser(userID, page, limit)
}

func (s *InstallationService) CreateBooking(userID uuid.UUID, input CreateBookingInput) (*models.InstallationBooking, error) {
	if input.ScheduledAt.Before(time.Now()) {
		return nil, ErrScheduledInPast
	}
	_ = s.installRepo.ExpireQuotesBefore(time.Now())

	quote, err := s.GetQuoteForUser(userID, input.QuoteID)
	if err != nil {
		return nil, err
	}
	if quote.Status != models.QuoteStatusReady {
		return nil, ErrInstallationQuoteNotReady
	}

	exists, err := s.installRepo.BookingExistsForQuote(quote.ID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("booking already exists for this quote")
	}

	var selectedLine *models.InstallationQuoteLine
	for i := range quote.Lines {
		if quote.Lines[i].ID == input.QuoteLineID {
			selectedLine = &quote.Lines[i]
			break
		}
	}
	if selectedLine == nil {
		return nil, ErrInstallationQuoteLineNotFound
	}

	profile, err := s.mechanicRepo.GetProfileByID(selectedLine.MechanicProfileID)
	if err != nil {
		return nil, err
	}

	partsTotal := 0.0
	if quote.OrderID != nil {
		order, err := s.orderRepo.GetByID(*quote.OrderID)
		if err == nil && order != nil {
			partsTotal = order.Subtotal
		}
	}

	laborTotal := selectedLine.LaborPrice
	platformFee := laborTotal * platformFeePercent
	total := partsTotal + laborTotal + platformFee

	booking := &models.InstallationBooking{
		QuoteID:           quote.ID,
		QuoteLineID:       selectedLine.ID,
		UserID:            userID,
		MechanicProfileID: profile.ID,
		MechanicUserID:    profile.UserID,
		Status:            models.BookingStatusPendingPayment,
		ScheduledAt:       input.ScheduledAt,
		ServiceStreet:     quote.ServiceStreet,
		ServiceCity:       quote.ServiceCity,
		ServiceState:      quote.ServiceState,
		ServicePostalCode: quote.ServicePostalCode,
		ServiceCountry:    quote.ServiceCountry,
		LaborTotal:        laborTotal,
		PartsTotal:        partsTotal,
		PlatformFee:       platformFee,
		TotalAmount:       total,
		PaymentStatus:     models.PaymentPending,
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(booking).Error; err != nil {
			return err
		}
		payment := &models.BookingPayment{
			BookingID: booking.ID,
			Provider:  "manual",
			Amount:    total,
			Status:    models.PaymentPending,
		}
		if err := tx.Create(payment).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.InstallationQuote{}).Where("id = ?", quote.ID).
			Update("status", models.QuoteStatusBooked).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.InstallationQuoteLine{}).Where("quote_id = ?", quote.ID).
			Update("status", models.QuoteLineStatusDeclined).Error; err != nil {
			return err
		}
		return tx.Model(&models.InstallationQuoteLine{}).Where("id = ?", selectedLine.ID).
			Update("status", models.QuoteLineStatusSelected).Error
	})
	if err != nil {
		return nil, err
	}

	loaded, err := s.installRepo.GetBookingByID(booking.ID)
	if err != nil {
		return nil, err
	}
	s.emitBookingConfirmed(userID, profile.UserID, booking.ID)
	return loaded, nil
}

func (s *InstallationService) GetBookingForUser(userID, bookingID uuid.UUID) (*models.InstallationBooking, error) {
	booking, err := s.installRepo.GetBookingByIDForUser(bookingID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInstallationBookingNotFound
		}
		return nil, err
	}
	return booking, nil
}

func (s *InstallationService) ListBookingsForUser(userID uuid.UUID, page, limit int) ([]models.InstallationBooking, int64, error) {
	return s.installRepo.ListBookingsForUser(userID, page, limit)
}

func (s *InstallationService) CancelBooking(userID, bookingID uuid.UUID, reason string) (*models.InstallationBooking, error) {
	booking, err := s.GetBookingForUser(userID, bookingID)
	if err != nil {
		return nil, err
	}
	if booking.Status == models.BookingStatusCompleted || booking.Status == models.BookingStatusCancelled {
		return nil, ErrBookingNotCancellable
	}
	booking.Status = models.BookingStatusCancelled
	booking.CancellationReason = reason
	booking.PaymentStatus = models.PaymentRefunded
	if err := s.installRepo.UpdateBooking(booking); err != nil {
		return nil, err
	}
	return booking, nil
}

func (s *InstallationService) ListQuoteLinesForMechanic(mechanicProfileID uuid.UUID, page, limit int) ([]models.InstallationQuoteLine, int64, error) {
	return s.installRepo.ListQuoteLinesForMechanic(mechanicProfileID, page, limit)
}

func (s *InstallationService) RespondToQuoteLine(mechanicProfileID, lineID uuid.UUID, input RespondQuoteLineInput) (*models.InstallationQuoteLine, error) {
	line, err := s.installRepo.GetQuoteLineByID(lineID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInstallationQuoteLineNotFound
		}
		return nil, err
	}
	if line.MechanicProfileID != mechanicProfileID {
		return nil, ErrQuoteLineNotForMechanic
	}

	quote, err := s.installRepo.GetQuoteByID(line.QuoteID)
	if err != nil {
		return nil, err
	}
	if quote.Status != models.QuoteStatusReady || time.Now().After(quote.ExpiresAt) {
		return nil, ErrInstallationQuoteExpired
	}

	if input.LaborPrice != nil && *input.LaborPrice > 0 {
		line.LaborPrice = *input.LaborPrice
	}
	if input.EstimatedHours != nil && *input.EstimatedHours > 0 {
		line.EstimatedHours = *input.EstimatedHours
	}
	if input.MechanicMessage != nil {
		line.MechanicMessage = *input.MechanicMessage
	}
	if err := s.installRepo.UpdateQuoteLine(line); err != nil {
		return nil, err
	}
	return s.installRepo.GetQuoteLineByID(lineID)
}

func (s *InstallationService) SetMechanicInstallServices(mechanicProfileID uuid.UUID, jobTypeIDs []uuid.UUID) error {
	jobTypes, err := s.installRepo.ListJobTypes(true)
	if err != nil {
		return err
	}
	priceByJob := make(map[uuid.UUID]float64, len(jobTypes))
	for _, jt := range jobTypes {
		priceByJob[jt.ID] = jt.BaseLaborPrice
	}
	for _, jtID := range jobTypeIDs {
		price := priceByJob[jtID]
		if price == 0 {
			continue
		}
		if err := s.installRepo.UpsertInstallService(&models.MechanicInstallService{
			MechanicProfileID: mechanicProfileID,
			JobTypeID:         jtID,
			LaborPrice:        price,
			IsActive:          true,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *InstallationService) ListBookingsForMechanic(mechanicUserID uuid.UUID, page, limit int) ([]models.InstallationBooking, int64, error) {
	return s.installRepo.ListBookingsForMechanic(mechanicUserID, page, limit)
}

func (s *InstallationService) GetBookingForMechanic(mechanicUserID, bookingID uuid.UUID) (*models.InstallationBooking, error) {
	booking, err := s.installRepo.GetBookingByIDForMechanic(bookingID, mechanicUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInstallationBookingNotFound
		}
		return nil, err
	}
	return booking, nil
}

func (s *InstallationService) UpdateBookingStatus(mechanicUserID, bookingID uuid.UUID, status models.BookingStatus) (*models.InstallationBooking, error) {
	if !models.IsValidBookingStatus(status) {
		return nil, ErrInvalidBookingStatusTransition
	}
	booking, err := s.GetBookingForMechanic(mechanicUserID, bookingID)
	if err != nil {
		return nil, err
	}
	if !isAllowedStatusTransition(booking.Status, status) {
		return nil, ErrInvalidBookingStatusTransition
	}

	prev := booking.Status
	booking.Status = status
	if status == models.BookingStatusConfirmed && booking.PaymentStatus == models.PaymentPending {
		booking.PaymentStatus = models.PaymentPaid
	}
	if err := s.installRepo.UpdateBooking(booking); err != nil {
		return nil, err
	}

	if status == models.BookingStatusEnRoute && prev != models.BookingStatusEnRoute {
		s.emitMechanicEnRoute(booking.UserID, booking.ID)
	}
	return booking, nil
}

func isAllowedStatusTransition(from, to models.BookingStatus) bool {
	if from == to {
		return true
	}
	allowed := map[models.BookingStatus][]models.BookingStatus{
		models.BookingStatusPendingPayment: {models.BookingStatusConfirmed, models.BookingStatusCancelled},
		models.BookingStatusConfirmed:      {models.BookingStatusEnRoute, models.BookingStatusInProgress, models.BookingStatusCancelled},
		models.BookingStatusEnRoute:        {models.BookingStatusInProgress, models.BookingStatusCancelled},
		models.BookingStatusInProgress:     {models.BookingStatusCompleted, models.BookingStatusCancelled},
	}
	for _, next := range allowed[from] {
		if next == to {
			return true
		}
	}
	return false
}

func uniqueJobTypeIDs(items []models.InstallationQuoteItem) []uuid.UUID {
	seen := make(map[uuid.UUID]bool)
	var ids []uuid.UUID
	for _, item := range items {
		if !seen[item.JobTypeID] {
			seen[item.JobTypeID] = true
			ids = append(ids, item.JobTypeID)
		}
	}
	return ids
}

func (s *InstallationService) emitQuoteReady(userID, quoteID uuid.UUID) {
	if s.notifier == nil {
		return
	}
	if err := s.notifier.QuoteReady(context.Background(), userID, quoteID); err != nil {
		s.log.Warn("failed to send quote ready notification", zap.Error(err))
	}
}

func (s *InstallationService) emitBookingConfirmed(customerID, mechanicUserID, bookingID uuid.UUID) {
	if s.notifier == nil {
		return
	}
	if err := s.notifier.BookingConfirmed(context.Background(), customerID, bookingID, false); err != nil {
		s.log.Warn("failed to send booking confirmed notification to customer", zap.Error(err))
	}
	if err := s.notifier.BookingConfirmed(context.Background(), mechanicUserID, bookingID, true); err != nil {
		s.log.Warn("failed to send booking confirmed notification to mechanic", zap.Error(err))
	}
}

func (s *InstallationService) emitMechanicEnRoute(customerID, bookingID uuid.UUID) {
	if s.notifier == nil {
		return
	}
	if err := s.notifier.MechanicEnRoute(context.Background(), customerID, bookingID); err != nil {
		s.log.Warn("failed to send mechanic en route notification", zap.Error(err))
	}
}
