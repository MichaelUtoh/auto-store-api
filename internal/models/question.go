package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type QuestionStatus string

const (
	QuestionStatusOpen     QuestionStatus = "open"
	QuestionStatusAnswered QuestionStatus = "answered"
	QuestionStatusClosed   QuestionStatus = "closed"
)

func IsValidQuestionStatus(s QuestionStatus) bool {
	switch s {
	case QuestionStatusOpen, QuestionStatusAnswered, QuestionStatusClosed:
		return true
	default:
		return false
	}
}

// Question is a community Q&A thread tied to catalog or vehicle context.
type Question struct {
	ID         uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	UserID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	Title      string         `gorm:"not null" json:"title"`
	Body       string         `gorm:"type:text;not null" json:"body"`
	Slug       string         `gorm:"uniqueIndex;not null" json:"slug"`
	ProductID  *uuid.UUID     `gorm:"type:uuid;index" json:"product_id,omitempty"`
	CategoryID *uuid.UUID     `gorm:"type:uuid;index" json:"category_id,omitempty"`
	Make       string         `gorm:"size:100" json:"make,omitempty"`
	Model      string         `gorm:"size:100" json:"model,omitempty"`
	Year       *int           `json:"year,omitempty"`
	Status      QuestionStatus `gorm:"type:varchar(20);not null;default:'open';index" json:"status"`
	ViewCount   int            `gorm:"column:view_count;default:0" json:"view_count"`
	AnswerCount int            `gorm:"-" json:"answer_count,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	User     User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Product  *Product  `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Category *Category `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Answers  []Answer  `gorm:"foreignKey:QuestionID" json:"answers,omitempty"`
}

func (Question) TableName() string { return "questions" }

func (q *Question) BeforeCreate(tx *gorm.DB) error {
	if q.ID == uuid.Nil {
		q.ID = uuid.New()
	}
	if q.Status == "" {
		q.Status = QuestionStatusOpen
	}
	return nil
}

// Answer is a response to a question; verified mechanics set IsVerifiedMechanic.
type Answer struct {
	ID                  uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	QuestionID          uuid.UUID      `gorm:"type:uuid;not null;index" json:"question_id"`
	UserID              uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	Body                string         `gorm:"type:text;not null" json:"body"`
	IsAccepted          bool           `gorm:"column:is_accepted;default:false" json:"is_accepted"`
	IsVerifiedMechanic  bool           `gorm:"column:is_verified_mechanic;default:false" json:"is_verified_mechanic"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`

	User     User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Question Question `gorm:"foreignKey:QuestionID" json:"-"`
}

func (Answer) TableName() string { return "answers" }

func (a *Answer) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
