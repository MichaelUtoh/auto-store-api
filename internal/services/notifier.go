package services

import (
	"context"
	"fmt"

	"auto-store-api/internal/models"
	"auto-store-api/pkg/email"

	"github.com/google/uuid"
)

// Notifier provides typed helpers for domain events.
type Notifier struct {
	notify *NotificationService
	appURL string
	email  email.Sender
}

func NewNotifier(notify *NotificationService, frontendURL string, emailSender email.Sender) *Notifier {
	return &Notifier{notify: notify, appURL: frontendURL, email: emailSender}
}

func (n *Notifier) MechanicVerified(ctx context.Context, userID uuid.UUID, businessName string) error {
	return n.notify.Notify(ctx, NotifyInput{
		UserID:         userID,
		Type:           models.NotificationMechanicVerified,
		IdempotencyKey: fmt.Sprintf("mechanic:%s:verified", userID),
		Title:          "Mechanic profile verified",
		Body:           fmt.Sprintf("Your mechanic profile for %s is now verified. You can accept installation jobs when that feature launches.", businessName),
		Payload: map[string]interface{}{
			"href": "/mechanic/profile",
		},
	})
}

func (n *Notifier) MechanicApplyReceived(ctx context.Context, userID uuid.UUID, businessName string) error {
	return n.notify.Notify(ctx, NotifyInput{
		UserID:         userID,
		Type:           models.NotificationMechanicApplyReceived,
		IdempotencyKey: fmt.Sprintf("mechanic:%s:apply_received", userID),
		Title:          "Application received",
		Body:           fmt.Sprintf("We received your mechanic application for %s. We'll notify you when it's reviewed.", businessName),
		Payload: map[string]interface{}{
			"href": "/mechanic/profile",
		},
	})
}

func (n *Notifier) QuoteReady(ctx context.Context, userID uuid.UUID, quoteID uuid.UUID) error {
	return n.notify.Notify(ctx, NotifyInput{
		UserID:         userID,
		Type:           models.NotificationQuoteReady,
		IdempotencyKey: fmt.Sprintf("quote:%s:ready", quoteID),
		Title:          "Installation quote ready",
		Body:           "Your installation quotes are ready to review.",
		Payload: map[string]interface{}{
			"quote_id": quoteID.String(),
			"href":     fmt.Sprintf("/installations/quotes/%s", quoteID),
		},
	})
}

func (n *Notifier) BookingConfirmed(ctx context.Context, userID uuid.UUID, bookingID uuid.UUID, forMechanic bool) error {
	title := "Installation booked"
	body := "Your installation appointment is confirmed."
	href := fmt.Sprintf("/installations/bookings/%s", bookingID)
	key := fmt.Sprintf("booking:%s:confirmed:customer", bookingID)
	if forMechanic {
		title = "New installation booking"
		body = "You have a new confirmed installation appointment."
		href = fmt.Sprintf("/mechanic/bookings/%s", bookingID)
		key = fmt.Sprintf("booking:%s:confirmed:mechanic:%s", bookingID, userID)
	}
	return n.notify.Notify(ctx, NotifyInput{
		UserID:         userID,
		Type:           models.NotificationBookingConfirmed,
		IdempotencyKey: key,
		Title:          title,
		Body:           body,
		Payload: map[string]interface{}{
			"booking_id": bookingID.String(),
			"href":       href,
		},
	})
}

func (n *Notifier) MechanicEnRoute(ctx context.Context, userID uuid.UUID, bookingID uuid.UUID) error {
	return n.notify.Notify(ctx, NotifyInput{
		UserID:         userID,
		Type:           models.NotificationMechanicEnRoute,
		IdempotencyKey: fmt.Sprintf("booking:%s:en_route", bookingID),
		Title:          "Mechanic is on the way",
		Body:           "Your mechanic is en route to your location.",
		Payload: map[string]interface{}{
			"booking_id": bookingID.String(),
			"href":       fmt.Sprintf("/installations/bookings/%s", bookingID),
		},
	})
}

func (n *Notifier) QAAnswerPosted(ctx context.Context, userID uuid.UUID, questionID uuid.UUID, slug string) error {
	href := fmt.Sprintf("/q/%s", slug)
	if slug == "" {
		href = fmt.Sprintf("/q/%s", questionID)
	}
	return n.notify.Notify(ctx, NotifyInput{
		UserID:         userID,
		Type:           models.NotificationQAAnswerPosted,
		IdempotencyKey: fmt.Sprintf("question:%s:answer", questionID),
		Title:          "Your question was answered",
		Body:           "A verified mechanic answered your question.",
		Payload: map[string]interface{}{
			"question_id": questionID.String(),
			"href":        href,
		},
	})
}

func (n *Notifier) SupportAdminReplied(ctx context.Context, conv *models.Conversation, messageID uuid.UUID, preview string) error {
	if conv == nil {
		return nil
	}
	href := "/support"
	if n.appURL != "" {
		href = n.appURL + href
	}
	title := "Support replied"
	body := fmt.Sprintf("Support replied to your message: %s", truncatePreview(preview, 120))

	if conv.UserID != nil {
		return n.notify.Notify(ctx, NotifyInput{
			UserID:         *conv.UserID,
			Type:           models.NotificationSupportAdminReplied,
			IdempotencyKey: fmt.Sprintf("support:%s:msg:%s:admin_replied", conv.ID, messageID),
			Title:          title,
			Body:           body,
			Payload: map[string]interface{}{
				"conversation_id": conv.ID.String(),
				"href":            "/support",
			},
		})
	}

	if conv.GuestEmail != "" && n.email != nil {
		emailBody := body
		if n.appURL != "" {
			emailBody += fmt.Sprintf("\n\nOpen chat: %s", href)
		}
		return n.email.Send(conv.GuestEmail, title, emailBody)
	}
	return nil
}

func (n *Notifier) SupportNewConversation(ctx context.Context, adminUserID uuid.UUID, conv *models.Conversation, preview string) error {
	if conv == nil {
		return nil
	}
	label := "Customer"
	if conv.GuestID != nil {
		if conv.GuestName != "" {
			label = conv.GuestName
		} else {
			label = "Guest"
		}
	}
	return n.notify.Notify(ctx, NotifyInput{
		UserID:         adminUserID,
		Type:           models.NotificationSupportNewConversation,
		IdempotencyKey: fmt.Sprintf("support:%s:new:admin:%s", conv.ID, adminUserID),
		Title:          "New support message",
		Body:           fmt.Sprintf("%s: %s", label, truncatePreview(preview, 120)),
		Payload: map[string]interface{}{
			"conversation_id": conv.ID.String(),
			"href":            fmt.Sprintf("/admin/support/%s", conv.ID),
		},
	})
}

func truncatePreview(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
