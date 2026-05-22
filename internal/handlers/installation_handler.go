package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"auto-store-api/internal/handlers/dto"
	"auto-store-api/internal/models"
	"auto-store-api/internal/services"
	"auto-store-api/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type InstallationHandler struct {
	install *services.InstallationService
}

func NewInstallationHandler(install *services.InstallationService) *InstallationHandler {
	return &InstallationHandler{install: install}
}

func (h *InstallationHandler) ListJobTypes(c *gin.Context) {
	types, err := h.install.ListJobTypes()
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	resp := make([]dto.InstallationJobTypeResponse, len(types))
	for i := range types {
		resp[i] = dto.InstallationJobTypeToResponse(&types[i])
	}
	utils.JSON(c, http.StatusOK, resp)
}

func (h *InstallationHandler) CreateQuote(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	var req dto.CreateInstallationQuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	if req.OrderID == nil && len(req.ProductIDs) == 0 {
		utils.JSONBadRequest(c, "order_id or product_ids is required")
		return
	}

	quote, err := h.install.CreateQuote(userID, services.CreateQuoteInput{
		OrderID:           req.OrderID,
		ProductIDs:        req.ProductIDs,
		VehicleMake:       req.VehicleMake,
		VehicleModel:      req.VehicleModel,
		VehicleYear:       req.VehicleYear,
		ServiceStreet:     req.ServiceStreet,
		ServiceCity:       req.ServiceCity,
		ServiceState:      req.ServiceState,
		ServicePostalCode: req.ServicePostalCode,
		ServiceCountry:    req.ServiceCountry,
		Latitude:          req.Latitude,
		Longitude:         req.Longitude,
		Notes:             req.Notes,
		SearchRadiusKm:    req.SearchRadiusKm,
	})
	if err != nil {
		h.handleInstallError(c, err)
		return
	}
	utils.JSON(c, http.StatusCreated, dto.InstallationQuoteToResponse(quote))
}

func (h *InstallationHandler) GetQuote(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	quoteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid quote id")
		return
	}
	quote, err := h.install.GetQuoteForUser(userID, quoteID)
	if err != nil {
		h.handleInstallError(c, err)
		return
	}
	utils.JSON(c, http.StatusOK, dto.InstallationQuoteToResponse(quote))
}

func (h *InstallationHandler) ListQuotes(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	quotes, total, err := h.install.ListQuotesForUser(userID, page, limit)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	resp := make([]dto.InstallationQuoteResponse, len(quotes))
	for i := range quotes {
		resp[i] = dto.InstallationQuoteToResponse(&quotes[i])
	}
	utils.JSONPaginated(c, resp, page, limit, total)
}

func (h *InstallationHandler) CreateBooking(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	var req dto.CreateInstallationBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	booking, err := h.install.CreateBooking(userID, services.CreateBookingInput{
		QuoteID:     req.QuoteID,
		QuoteLineID: req.QuoteLineID,
		ScheduledAt: req.ScheduledAt,
	})
	if err != nil {
		h.handleInstallError(c, err)
		return
	}
	utils.JSON(c, http.StatusCreated, dto.InstallationBookingToResponse(booking))
}

func (h *InstallationHandler) GetBooking(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	bookingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid booking id")
		return
	}
	booking, err := h.install.GetBookingForUser(userID, bookingID)
	if err != nil {
		h.handleInstallError(c, err)
		return
	}
	utils.JSON(c, http.StatusOK, dto.InstallationBookingToResponse(booking))
}

func (h *InstallationHandler) ListBookings(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	bookings, total, err := h.install.ListBookingsForUser(userID, page, limit)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	resp := make([]dto.InstallationBookingResponse, len(bookings))
	for i := range bookings {
		resp[i] = dto.InstallationBookingToResponse(&bookings[i])
	}
	utils.JSONPaginated(c, resp, page, limit, total)
}

func (h *InstallationHandler) CancelBooking(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	bookingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid booking id")
		return
	}
	var req dto.CancelInstallationBookingRequest
	_ = c.ShouldBindJSON(&req)
	booking, err := h.install.CancelBooking(userID, bookingID, req.Reason)
	if err != nil {
		h.handleInstallError(c, err)
		return
	}
	utils.JSON(c, http.StatusOK, dto.InstallationBookingToResponse(booking))
}

func (h *InstallationHandler) ListMechanicQuoteLines(c *gin.Context) {
	profile := mechanicProfileFromContext(c)
	if profile == nil {
		utils.JSONForbidden(c, "verified mechanic profile required")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	lines, total, err := h.install.ListQuoteLinesForMechanic(profile.ID, page, limit)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	resp := make([]dto.InstallationQuoteLineResponse, len(lines))
	for i := range lines {
		name := ""
		rating := 0.0
		if lines[i].MechanicProfile.ID != uuid.Nil {
			name = lines[i].MechanicProfile.BusinessName
			rating = lines[i].MechanicProfile.RatingAvg
		}
		jobName := ""
		if lines[i].JobType.ID != uuid.Nil {
			jobName = lines[i].JobType.Name
		}
		resp[i] = dto.InstallationQuoteLineResponse{
			ID:                lines[i].ID,
			MechanicProfileID: lines[i].MechanicProfileID,
			MechanicName:      name,
			JobTypeID:         lines[i].JobTypeID,
			JobTypeName:       jobName,
			LaborPrice:        lines[i].LaborPrice,
			EstimatedHours:    lines[i].EstimatedHours,
			MechanicMessage:   lines[i].MechanicMessage,
			DistanceKm:        lines[i].DistanceKm,
			Status:            string(lines[i].Status),
			RatingAvg:         rating,
		}
	}
	utils.JSONPaginated(c, resp, page, limit, total)
}

func (h *InstallationHandler) RespondToQuoteLine(c *gin.Context) {
	profile := mechanicProfileFromContext(c)
	if profile == nil {
		utils.JSONForbidden(c, "verified mechanic profile required")
		return
	}
	lineID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid quote line id")
		return
	}
	var req dto.RespondInstallationQuoteLineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	line, err := h.install.RespondToQuoteLine(profile.ID, lineID, services.RespondQuoteLineInput{
		LaborPrice:      req.LaborPrice,
		EstimatedHours:  req.EstimatedHours,
		MechanicMessage: req.MechanicMessage,
	})
	if err != nil {
		h.handleInstallError(c, err)
		return
	}
	utils.JSON(c, http.StatusOK, gin.H{
		"id":               line.ID,
		"labor_price":      line.LaborPrice,
		"estimated_hours":  line.EstimatedHours,
		"mechanic_message": line.MechanicMessage,
		"status":           line.Status,
	})
}

func (h *InstallationHandler) SetMechanicInstallServices(c *gin.Context) {
	profile := mechanicProfileFromContext(c)
	if profile == nil {
		utils.JSONForbidden(c, "verified mechanic profile required")
		return
	}
	var req dto.MechanicInstallServicesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	if err := h.install.SetMechanicInstallServices(profile.ID, req.JobTypeIDs); err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, gin.H{"updated": len(req.JobTypeIDs)})
}

func (h *InstallationHandler) ListMechanicBookings(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	bookings, total, err := h.install.ListBookingsForMechanic(userID, page, limit)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	resp := make([]dto.InstallationBookingResponse, len(bookings))
	for i := range bookings {
		resp[i] = dto.InstallationBookingToResponse(&bookings[i])
	}
	utils.JSONPaginated(c, resp, page, limit, total)
}

func (h *InstallationHandler) GetMechanicBooking(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	bookingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid booking id")
		return
	}
	booking, err := h.install.GetBookingForMechanic(userID, bookingID)
	if err != nil {
		h.handleInstallError(c, err)
		return
	}
	utils.JSON(c, http.StatusOK, dto.InstallationBookingToResponse(booking))
}

func (h *InstallationHandler) UpdateMechanicBookingStatus(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	bookingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid booking id")
		return
	}
	var req dto.UpdateInstallationBookingStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	booking, err := h.install.UpdateBookingStatus(userID, bookingID, models.BookingStatus(req.Status))
	if err != nil {
		h.handleInstallError(c, err)
		return
	}
	utils.JSON(c, http.StatusOK, dto.InstallationBookingToResponse(booking))
}

func mechanicProfileFromContext(c *gin.Context) *models.MechanicProfile {
	val, ok := c.Get("mechanic_profile")
	if !ok {
		return nil
	}
	p, ok := val.(*models.MechanicProfile)
	if !ok {
		return nil
	}
	return p
}

func (h *InstallationHandler) handleInstallError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrInstallationQuoteNotFound),
		errors.Is(err, services.ErrInstallationBookingNotFound),
		errors.Is(err, services.ErrOrderNotFoundForInstall):
		utils.JSONNotFound(c, err.Error())
	case errors.Is(err, services.ErrInstallationQuoteExpired),
		errors.Is(err, services.ErrInstallationQuoteNotReady),
		errors.Is(err, services.ErrNoMechanicsInArea),
		errors.Is(err, services.ErrNoInstallableProducts),
		errors.Is(err, services.ErrScheduledInPast),
		errors.Is(err, services.ErrBookingNotCancellable),
		errors.Is(err, services.ErrInvalidBookingStatusTransition),
		errors.Is(err, services.ErrQuoteLineNotForMechanic):
		utils.JSONBadRequest(c, err.Error())
	case errors.Is(err, services.ErrOrderNotOwned):
		utils.JSONForbidden(c, err.Error())
	default:
		utils.JSONInternal(c, err.Error())
	}
}
