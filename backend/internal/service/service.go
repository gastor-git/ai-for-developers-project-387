package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"booking-calendar/backend/internal/model"
	"booking-calendar/backend/internal/store"
)

var (
	emailRe = regexp.MustCompile(`^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$`)
	dateRe  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	timeRe  = regexp.MustCompile(`^([01]\d|2[0-3]):([0-5]\d)$`)

	ErrNotFound = errors.New("not found")
)

type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string { return e.Message }

func badRequest(msg string) error {
	return &APIError{Status: 400, Code: "BAD_REQUEST", Message: msg}
}

func notFound(msg string) error {
	return &APIError{Status: 404, Code: "NOT_FOUND", Message: msg}
}

func conflict(msg string) error {
	return &APIError{Status: 409, Code: "CONFLICT", Message: msg}
}

type Service struct {
	store *store.Store
	loc   *time.Location
	now   func() time.Time
}

func New(st *store.Store, loc *time.Location, now func() time.Time) *Service {
	return &Service{store: st, loc: loc, now: now}
}

func (s *Service) GetOwner(ctx context.Context, id string) (*model.Owner, error) {
	owner, err := s.store.GetOwner(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, notFound("Владелец не найден")
		}
		return nil, err
	}
	return owner, nil
}

func (s *Service) UpdateSchedule(ctx context.Context, id string, schedule model.Schedule) (*model.Owner, error) {
	if err := validateSchedule(schedule); err != nil {
		return nil, err
	}
	if err := s.store.UpdateOwnerSchedule(ctx, id, schedule); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, notFound("Владелец не найден")
		}
		return nil, err
	}
	return s.GetOwner(ctx, id)
}

func (s *Service) ListEvents(ctx context.Context, ownerID string) ([]model.Event, error) {
	if err := s.ensureOwner(ctx, ownerID); err != nil {
		return nil, err
	}
	return s.store.ListEvents(ctx, ownerID)
}

func (s *Service) CreateEvent(ctx context.Context, ownerID string, ec model.EventCreate) (*model.Event, error) {
	if err := s.ensureOwner(ctx, ownerID); err != nil {
		return nil, err
	}
	title := strings.TrimSpace(ec.Title)
	if title == "" {
		return nil, badRequest("Название события не может быть пустым")
	}
	if err := validateDuration(ec.DurationMinutes); err != nil {
		return nil, err
	}
	id, err := newID()
	if err != nil {
		return nil, err
	}
	ec.Title = title
	ec.Description = strings.TrimSpace(ec.Description)
	return s.store.CreateEvent(ctx, ownerID, ec, id)
}

func (s *Service) GetEvent(ctx context.Context, ownerID, eventID string) (*model.Event, error) {
	ev, err := s.store.GetEvent(ctx, ownerID, eventID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, notFound("Событие не найдено")
		}
		return nil, err
	}
	return ev, nil
}

func (s *Service) Slots(ctx context.Context, ownerID, eventID, dateStr string) ([]model.Slot, error) {
	if !dateRe.MatchString(dateStr) {
		return nil, badRequest("Дата должна быть в формате YYYY-MM-DD")
	}
	date, err := time.ParseInLocation("2006-01-02", dateStr, s.loc)
	if err != nil {
		return nil, badRequest("Некорректная дата")
	}
	owner, err := s.GetOwner(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	ev, err := s.GetEvent(ctx, ownerID, eventID)
	if err != nil {
		return nil, err
	}

	now := s.now().In(s.loc)
	today := startOfLocalDay(now)
	if date.Before(today) || date.After(today.AddDate(0, 0, windowDays-1)) {
		return []model.Slot{}, nil
	}

	day := scheduleDay(owner.Schedule, date.Weekday())
	if !day.IsWorking {
		return []model.Slot{}, nil
	}
	startMin, endMin, ok := scheduleMinutes(day)
	if !ok || endMin <= startMin {
		return []model.Slot{}, nil
	}

	duration := time.Duration(ev.DurationMinutes) * time.Minute
	dayStart := startOfLocalDay(date)
	workStart := dayStart.Add(time.Duration(startMin) * time.Minute)
	workEnd := dayStart.Add(time.Duration(endMin) * time.Minute)

	bookings, err := s.store.BookingsOnDate(ctx, dateStr)
	if err != nil {
		return nil, err
	}

	slots := make([]model.Slot, 0)
	for slotStart := workStart; !slotStart.Add(duration).After(workEnd); slotStart = slotStart.Add(duration) {
		slotEnd := slotStart.Add(duration)
		if !slotStart.After(now) {
			continue
		}
		if overlapsAny(bookings, slotStart, slotEnd) {
			continue
		}
		slots = append(slots, model.Slot{
			StartAt: slotStart.UTC(),
			EndAt:   slotEnd.UTC(),
		})
	}
	return slots, nil
}

func (s *Service) CreateBooking(ctx context.Context, ownerID, eventID string, bc model.BookingCreate) (*model.Booking, error) {
	owner, err := s.GetOwner(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	ev, err := s.GetEvent(ctx, ownerID, eventID)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(bc.Name)
	if name == "" {
		return nil, badRequest("Укажите имя гостя")
	}
	email := strings.TrimSpace(bc.Email)
	if !emailRe.MatchString(email) {
		return nil, badRequest("Укажите корректный email")
	}
	startAt := bc.StartAt
	if startAt.IsZero() {
		return nil, badRequest("Укажите время начала бронирования")
	}

	now := s.now()
	if !startAt.After(now) {
		return nil, badRequest("Слот должен начинаться в будущем")
	}

	nowLocal := now.In(s.loc)
	local := startAt.In(s.loc)
	today := startOfLocalDay(nowLocal)
	dayStart := startOfLocalDay(local)
	if dayStart.Before(today) || dayStart.After(today.AddDate(0, 0, windowDays-1)) {
		return nil, badRequest("Выбранный слот вне окна записи")
	}

	day := scheduleDay(owner.Schedule, local.Weekday())
	if !day.IsWorking {
		return nil, badRequest("Выбранный день — выходной")
	}
	startMin, endMin, ok := scheduleMinutes(day)
	if !ok || endMin <= startMin {
		return nil, badRequest("Некорректный график работы")
	}

	duration := ev.DurationMinutes
	mins := local.Hour()*60 + local.Minute()
	if mins < startMin || mins+duration > endMin {
		return nil, badRequest("Слот не умещается в рабочее время")
	}
	if (mins-startMin)%duration != 0 {
		return nil, badRequest("Слот не соответствует сетке расписания")
	}

	booking := &model.Booking{
		EventID:    ev.ID,
		Date:       dayStart.Format("2006-01-02"),
		StartAt:    startAt.UTC(),
		EndAt:      startAt.Add(time.Duration(duration) * time.Minute).UTC(),
		GuestName:  name,
		GuestEmail: email,
		CreatedAt:  now.UTC(),
	}
	id, err := newID()
	if err != nil {
		return nil, err
	}
	booking.ID = id

	if err := s.store.CreateBooking(ctx, booking); err != nil {
		if errors.Is(err, store.ErrConflict) {
			return nil, conflict("Выбранный слот уже занят. Выберите другой.")
		}
		return nil, err
	}
	return booking, nil
}

func (s *Service) ListBookings(ctx context.Context, ownerID string) ([]model.Booking, error) {
	if err := s.ensureOwner(ctx, ownerID); err != nil {
		return nil, err
	}
	return s.store.ListBookings(ctx, ownerID)
}

func (s *Service) ensureOwner(ctx context.Context, id string) error {
	_, err := s.store.GetOwner(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return notFound("Владелец не найден")
		}
		return err
	}
	return nil
}

const windowDays = 14

func startOfLocalDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func scheduleDay(sch model.Schedule, wd time.Weekday) model.DaySchedule {
	switch wd {
	case time.Monday:
		return sch.Monday
	case time.Tuesday:
		return sch.Tuesday
	case time.Wednesday:
		return sch.Wednesday
	case time.Thursday:
		return sch.Thursday
	case time.Friday:
		return sch.Friday
	case time.Saturday:
		return sch.Saturday
	default:
		return sch.Sunday
	}
}

func scheduleMinutes(day model.DaySchedule) (int, int, bool) {
	sh, sm, sok := parseHHMM(day.Start)
	eh, em, eok := parseHHMM(day.End)
	if !sok || !eok {
		return 0, 0, false
	}
	return sh*60 + sm, eh*60 + em, true
}

func parseHHMM(v string) (int, int, bool) {
	m := timeRe.FindStringSubmatch(v)
	if m == nil {
		return 0, 0, false
	}
	h, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, 0, false
	}
	min, err := strconv.Atoi(m[2])
	if err != nil {
		return 0, 0, false
	}
	return h, min, true
}

func validateSchedule(sch model.Schedule) error {
	for _, day := range []model.DaySchedule{sch.Monday, sch.Tuesday, sch.Wednesday, sch.Thursday, sch.Friday, sch.Saturday, sch.Sunday} {
		sh, sm, sok := parseHHMM(day.Start)
		eh, em, eok := parseHHMM(day.End)
		if !sok || !eok {
			return badRequest("Время начала и конца рабочего дня указывается в формате HH:MM")
		}
		if day.IsWorking && eh*60+em <= sh*60+sm {
			return badRequest("Конец рабочего дня должен быть позже начала")
		}
	}
	return nil
}

func validateDuration(d int) error {
	if d < 15 || d > 480 {
		return badRequest("Длительность события должна быть целым числом от 15 до 480 минут")
	}
	return nil
}

func overlapsAny(bookings []model.Booking, start, end time.Time) bool {
	for _, b := range bookings {
		if start.Before(b.EndAt) && end.After(b.StartAt) {
			return true
		}
	}
	return false
}

func newID() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
