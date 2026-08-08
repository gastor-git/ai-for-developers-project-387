package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"booking-calendar/backend/internal/model"
	"booking-calendar/backend/internal/service"
)

type API struct {
	svc *service.Service
}

func NewHandler(svc *service.Service) http.Handler {
	api := &API{svc: svc}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", api.health)
	mux.HandleFunc("GET /api/owners/{ownerId}", api.getOwner)
	mux.HandleFunc("PATCH /api/owners/{ownerId}/schedule", api.updateSchedule)
	mux.HandleFunc("GET /api/owners/{ownerId}/events", api.listEvents)
	mux.HandleFunc("POST /api/owners/{ownerId}/events", api.createEvent)
	mux.HandleFunc("GET /api/owners/{ownerId}/events/{eventId}", api.getEvent)
	mux.HandleFunc("GET /api/owners/{ownerId}/events/{eventId}/slots", api.getSlots)
	mux.HandleFunc("POST /api/owners/{ownerId}/events/{eventId}/bookings", api.createBooking)
	mux.HandleFunc("GET /api/owners/{ownerId}/bookings", api.listBookings)

	return withMiddleware(mux)
}

func (a *API) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) getOwner(w http.ResponseWriter, r *http.Request) {
	owner, err := a.svc.GetOwner(r.Context(), r.PathValue("ownerId"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, owner)
}

func (a *API) updateSchedule(w http.ResponseWriter, r *http.Request) {
	var schedule model.Schedule
	if err := decodeJSON(w, r, &schedule); err != nil {
		writeError(w, err)
		return
	}
	owner, err := a.svc.UpdateSchedule(r.Context(), r.PathValue("ownerId"), schedule)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, owner)
}

func (a *API) listEvents(w http.ResponseWriter, r *http.Request) {
	events, err := a.svc.ListEvents(r.Context(), r.PathValue("ownerId"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (a *API) createEvent(w http.ResponseWriter, r *http.Request) {
	var body model.EventCreate
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, err)
		return
	}
	ev, err := a.svc.CreateEvent(r.Context(), r.PathValue("ownerId"), body)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ev)
}

func (a *API) getEvent(w http.ResponseWriter, r *http.Request) {
	ev, err := a.svc.GetEvent(r.Context(), r.PathValue("ownerId"), r.PathValue("eventId"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ev)
}

func (a *API) getSlots(w http.ResponseWriter, r *http.Request) {
	slots, err := a.svc.Slots(r.Context(), r.PathValue("ownerId"), r.PathValue("eventId"), r.URL.Query().Get("date"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, slots)
}

func (a *API) createBooking(w http.ResponseWriter, r *http.Request) {
	var body model.BookingCreate
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, err)
		return
	}
	booking, err := a.svc.CreateBooking(r.Context(), r.PathValue("ownerId"), r.PathValue("eventId"), body)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, booking)
}

func (a *API) listBookings(w http.ResponseWriter, r *http.Request) {
	bookings, err := a.svc.ListBookings(r.Context(), r.PathValue("ownerId"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bookings)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		return serviceBadRequest("Некорректное тело запроса")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

func writeError(w http.ResponseWriter, err error) {
	var ae *service.APIError
	if errors.As(err, &ae) {
		writeJSON(w, ae.Status, map[string]any{
			"error": map[string]string{"code": ae.Code, "message": ae.Message},
		})
		return
	}
	log.Printf("internal error: %v", err)
	writeJSON(w, http.StatusInternalServerError, map[string]any{
		"error": map[string]string{"code": "INTERNAL", "message": "Внутренняя ошибка сервера"},
	})
}

func serviceBadRequest(msg string) error {
	return &service.APIError{Status: 400, Code: "BAD_REQUEST", Message: msg}
}
