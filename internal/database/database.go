package database

import (
	"auto-store-api/internal/models"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the global database instance
var DB *gorm.DB

const (
	defaultRetryAttempts = 10
	defaultRetryDelay    = 2 * time.Second
)

// Connect establishes database connection and runs migrations
func Connect(dsn string, log *zap.Logger) error {
	return connect(dsn, log, 1, 0)
}

// ConnectWithRetry tries Connect with backoff (for Docker when Postgres may not be ready)
func ConnectWithRetry(dsn string, log *zap.Logger, attempts int, delay time.Duration) error {
	if attempts <= 0 {
		attempts = defaultRetryAttempts
	}
	if delay <= 0 {
		delay = defaultRetryDelay
	}
	return connect(dsn, log, attempts, delay)
}

func connect(dsn string, log *zap.Logger, attempts int, delay time.Duration) error {
	var err error
	for attempt := 1; attempt <= attempts; attempt++ {
		err = doConnect(dsn, log)
		if err == nil {
			return nil
		}
		if attempt < attempts {
			log.Warn("database connection failed, retrying",
				zap.Int("attempt", attempt),
				zap.Int("max", attempts),
				zap.Duration("next_in", delay),
				zap.Error(err))
			time.Sleep(delay)
		}
	}
	return err
}

func doConnect(dsn string, log *zap.Logger) error {
	var err error
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}
	DB, err = gorm.Open(postgres.Open(dsn), config)
	if err != nil {
		return err
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	if err := AutoMigrate(DB); err != nil {
		log.Error("migration failed", zap.Error(err))
		return err
	}
	log.Info("database connected and migrated")
	return nil
}

// AutoMigrate runs GORM AutoMigrate for all models
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.User{},
		&models.Address{},
		&models.Product{},
		&models.Category{},
		&models.ProductCategory{},
		&models.Tag{},
		&models.ProductTag{},
		&models.ProductImage{},
		&models.Specification{},
		&models.VehicleCompatibility{},
		&models.Order{},
		&models.OrderItem{},
		&models.Review{},
		&models.CartItem{},
		&models.WishlistItem{},
		&models.PasswordResetToken{},
		&models.EmailVerificationToken{},
		&models.MechanicProfile{},
		&models.MechanicDocument{},
		&models.Notification{},
		&models.NotificationPreference{},
		&models.InstallationJobType{},
		&models.MechanicInstallService{},
		&models.InstallationQuote{},
		&models.InstallationQuoteItem{},
		&models.InstallationQuoteLine{},
		&models.InstallationBooking{},
		&models.BookingPayment{},
	); err != nil {
		return err
	}
	return SeedInstallationJobTypes(db)
}
