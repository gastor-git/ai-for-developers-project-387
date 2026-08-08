package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"booking-calendar/backend/internal/model"
	"booking-calendar/backend/internal/store"
)

var (
	testLoc = time.UTC
	// Понедельник, 2026-08-10 08:00 UTC. График по умолчанию: Пн–Пт 09:00–18:00.
	testNow = time.Date(2026, 8, 10, 8, 0, 0, 0, time.UTC)
)

func testOwner() model.Owner {
	return model.Owner{
		ID:   "1",
		Name: "Владелец календаря",
		Schedule: model.Schedule{
			Monday:    model.DaySchedule{IsWorking: true, Start: "09:00", End: "18:00"},
			Tuesday:   model.DaySchedule{IsWorking: true, Start: "09:00", End: "18:00"},
			Wednesday: model.DaySchedule{IsWorking: true, Start: "09:00", End: "18:00"},
			Thursday:  model.DaySchedule{IsWorking: true, Start: "09:00", End: "18:00"},
			Friday:    model.DaySchedule{IsWorking: true, Start: "09:00", End: "18:00"},
			Saturday:  model.DaySchedule{IsWorking: false, Start: "09:00", End: "18:00"},
			Sunday:    model.DaySchedule{IsWorking: false, Start: "09:00", End: "18:00"},
		},
	}
}

func newTestEnv(t *testing.T, now time.Time) (*Service, *store.Store) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := st.SeedOwner(context.Background(), testOwner()); err != nil {
		t.Fatalf("seed owner: %v", err)
	}
	return New(st, testLoc, func() time.Time { return now }), st
}

func createEvent(t *testing.T, svc *Service, duration int) *model.Event {
	t.Helper()
	ev, err := svc.CreateEvent(context.Background(), "1", model.EventCreate{
		Title:           "Тест",
		Description:     "Описание",
		DurationMinutes: duration,
	})
	if err != nil {
		t.Fatalf("create event: %v", err)
	}
	return ev
}

func book(t *testing.T, svc *Service, eventID string, start time.Time) *model.Booking {
	t.Helper()
	b, err := svc.CreateBooking(context.Background(), "1", eventID, model.BookingCreate{
		Name:    "Иван",
		Email:   "ivan@example.com",
		StartAt: start,
	})
	if err != nil {
		t.Fatalf("create booking: %v", err)
	}
	return b
}

func expectStatus(t *testing.T, err error, want int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	var ae *APIError
	if !errors.As(err, &ae) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if ae.Status != want {
		t.Fatalf("expected status %d, got %d (%s)", want, ae.Status, ae.Message)
	}
}

func hhmm(t time.Time) string {
	return t.In(testLoc).Format("15:04")
}

func utcDate(y int, m time.Month, d int, h, min int) time.Time {
	return time.Date(y, m, d, h, min, 0, 0, time.UTC)
}

// Окно записи: [сегодня, сегодня+13]. Дата вне окна — пустой список,
// невалидная строка даты — ошибка 400.
func TestSlotsWindow(t *testing.T) {
	svc, _ := newTestEnv(t, testNow)
	ev := createEvent(t, svc, 60)
	ctx := context.Background()

	today := testNow.Format("2006-01-02")
	slots, err := svc.Slots(ctx, "1", ev.ID, today)
	if err != nil {
		t.Fatalf("slots for today: %v", err)
	}
	if len(slots) == 0 {
		t.Fatalf("expected slots for today %s, got none", today)
	}

	before := utcDate(2026, 8, 9, 0, 0).Format("2006-01-02")
	slots, err = svc.Slots(ctx, "1", ev.ID, before)
	if err != nil {
		t.Fatalf("slots before window: %v", err)
	}
	if len(slots) != 0 {
		t.Fatalf("expected no slots before window, got %d", len(slots))
	}

	after := utcDate(2026, 8, 24, 0, 0).Format("2006-01-02")
	slots, err = svc.Slots(ctx, "1", ev.ID, after)
	if err != nil {
		t.Fatalf("slots after window: %v", err)
	}
	if len(slots) != 0 {
		t.Fatalf("expected no slots after window, got %d", len(slots))
	}

	_, err = svc.Slots(ctx, "1", ev.ID, "2026/08/10")
	expectStatus(t, err, 400)

	_, err = svc.Slots(ctx, "1", ev.ID, "2026-02-30")
	expectStatus(t, err, 400)
}

// Выходные дни из графика не генерируют слоты.
func TestSlotsWeekend(t *testing.T) {
	svc, _ := newTestEnv(t, testNow)
	ev := createEvent(t, svc, 60)
	ctx := context.Background()

	for _, date := range []string{"2026-08-15", "2026-08-16"} {
		slots, err := svc.Slots(ctx, "1", ev.ID, date)
		if err != nil {
			t.Fatalf("slots on %s: %v", date, err)
		}
		if len(slots) != 0 {
			t.Fatalf("expected no slots on %s (weekend), got %d", date, len(slots))
		}
	}
}

// Слоты, начавшиеся в прошлом, исключаются: при now = 10:30 первый слот 11:00.
func TestSlotsPast(t *testing.T) {
	now := utcDate(2026, 8, 10, 10, 30)
	svc, _ := newTestEnv(t, now)
	ev := createEvent(t, svc, 60)

	slots, err := svc.Slots(context.Background(), "1", ev.ID, "2026-08-10")
	if err != nil {
		t.Fatalf("slots: %v", err)
	}
	if len(slots) == 0 {
		t.Fatal("expected slots, got none")
	}
	if got := hhmm(slots[0].StartAt); got != "11:00" {
		t.Fatalf("expected first slot 11:00, got %s", got)
	}
	for _, s := range slots {
		if !s.StartAt.After(now) {
			t.Fatalf("slot %s must start in the future", s.StartAt)
		}
	}
}

// Шаг сетки равен длительности события: старт каждого следующего слота сразу
// после конца предыдущего.
func TestSlotsGridStep(t *testing.T) {
	svc, _ := newTestEnv(t, testNow)
	ev := createEvent(t, svc, 40)

	slots, err := svc.Slots(context.Background(), "1", ev.ID, "2026-08-10")
	if err != nil {
		t.Fatalf("slots: %v", err)
	}
	if len(slots) == 0 {
		t.Fatal("expected slots, got none")
	}
	if got := hhmm(slots[0].StartAt); got != "09:00" {
		t.Fatalf("expected first slot 09:00, got %s", got)
	}
	for i := 1; i < len(slots); i++ {
		prevEnd := slots[i-1].StartAt.Add(40 * time.Minute)
		if !slots[i].StartAt.Equal(prevEnd) {
			t.Fatalf("slot %d start %s is not right after previous end %s",
				i, slots[i].StartAt, prevEnd)
		}
		if slots[i].EndAt.After(utcDate(2026, 8, 10, 18, 0)) {
			t.Fatalf("slot %d end %s exceeds working hours", i, slots[i].EndAt)
		}
	}
}

// Занятый слот исключается, соседние остаются свободными.
func TestSlotsOccupied(t *testing.T) {
	svc, _ := newTestEnv(t, testNow)
	ev := createEvent(t, svc, 60)
	book(t, svc, ev.ID, utcDate(2026, 8, 10, 10, 0))

	slots, err := svc.Slots(context.Background(), "1", ev.ID, "2026-08-10")
	if err != nil {
		t.Fatalf("slots: %v", err)
	}
	seen := map[string]bool{}
	for _, s := range slots {
		seen[hhmm(s.StartAt)] = true
	}
	if seen["10:00"] {
		t.Fatal("occupied slot 10:00 must be excluded")
	}
	if !seen["09:00"] {
		t.Fatal("expected free slot 09:00")
	}
	if !seen["11:00"] {
		t.Fatal("expected free slot 11:00")
	}
}

// Повторное бронирование того же слота — 409.
func TestCreateBookingConflictSameSlot(t *testing.T) {
	svc, _ := newTestEnv(t, testNow)
	ev := createEvent(t, svc, 60)
	start := utcDate(2026, 8, 10, 10, 0)

	book(t, svc, ev.ID, start)
	_, err := svc.CreateBooking(context.Background(), "1", ev.ID, model.BookingCreate{
		Name: "Пётр", Email: "petr@example.com", StartAt: start,
	})
	expectStatus(t, err, 409)
}

// Частичное наложение с бронированием другого события — тоже 409.
func TestCreateBookingConflictOverlapOtherEvent(t *testing.T) {
	svc, _ := newTestEnv(t, testNow)
	longEvent := createEvent(t, svc, 60)
	shortEvent := createEvent(t, svc, 30)

	book(t, svc, longEvent.ID, utcDate(2026, 8, 10, 10, 0))

	_, err := svc.CreateBooking(context.Background(), "1", shortEvent.ID, model.BookingCreate{
		Name: "Пётр", Email: "petr@example.com",
		StartAt: utcDate(2026, 8, 10, 10, 30),
	})
	expectStatus(t, err, 409)
}

// Смежные слоты (встык) бронируются без конфликта.
func TestCreateBookingAdjacentSlots(t *testing.T) {
	svc, _ := newTestEnv(t, testNow)
	ev := createEvent(t, svc, 30)
	book(t, svc, ev.ID, utcDate(2026, 8, 10, 9, 0))
	book(t, svc, ev.ID, utcDate(2026, 8, 10, 9, 30))
}

func TestCreateBookingValidation(t *testing.T) {
	svc, _ := newTestEnv(t, testNow)
	ev := createEvent(t, svc, 60)
	ctx := context.Background()
	start := utcDate(2026, 8, 10, 10, 0)

	base := model.BookingCreate{Name: "Иван", Email: "ivan@example.com", StartAt: start}

	cases := []struct {
		name string
		mut  func(*model.BookingCreate)
	}{
		{"empty name", func(b *model.BookingCreate) { b.Name = "  " }},
		{"bad email", func(b *model.BookingCreate) { b.Email = "not-an-email" }},
		{"missing email", func(b *model.BookingCreate) { b.Email = "" }},
		{"start in the past", func(b *model.BookingCreate) { b.StartAt = utcDate(2026, 8, 10, 7, 0) }},
		{"outside window", func(b *model.BookingCreate) { b.StartAt = utcDate(2026, 8, 24, 10, 0) }},
		{"weekend", func(b *model.BookingCreate) { b.StartAt = utcDate(2026, 8, 15, 10, 0) }},
		{"not on grid", func(b *model.BookingCreate) { b.StartAt = utcDate(2026, 8, 10, 9, 30) }},
		{"does not fit working hours", func(b *model.BookingCreate) { b.StartAt = utcDate(2026, 8, 10, 17, 30) }},
		{"zero startAt", func(b *model.BookingCreate) { b.StartAt = time.Time{} }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bc := base
			tc.mut(&bc)
			_, err := svc.CreateBooking(ctx, "1", ev.ID, bc)
			expectStatus(t, err, 400)
		})
	}
}

func TestCreateEventValidation(t *testing.T) {
	svc, _ := newTestEnv(t, testNow)
	ctx := context.Background()

	_, err := svc.CreateEvent(ctx, "1", model.EventCreate{Title: "", DurationMinutes: 30})
	expectStatus(t, err, 400)

	_, err = svc.CreateEvent(ctx, "1", model.EventCreate{Title: "X", DurationMinutes: 14})
	expectStatus(t, err, 400)

	_, err = svc.CreateEvent(ctx, "1", model.EventCreate{Title: "X", DurationMinutes: 481})
	expectStatus(t, err, 400)
}

func TestOwnerAndNotFound(t *testing.T) {
	svc, _ := newTestEnv(t, testNow)
	ctx := context.Background()

	owner, err := svc.GetOwner(ctx, "1")
	if err != nil {
		t.Fatalf("get owner: %v", err)
	}
	if owner.ID != "1" || owner.Name == "" {
		t.Fatalf("unexpected owner: %+v", owner)
	}
	if !owner.Schedule.Monday.IsWorking || owner.Schedule.Monday.Start != "09:00" {
		t.Fatalf("unexpected default schedule: %+v", owner.Schedule.Monday)
	}
	if owner.Schedule.Saturday.IsWorking {
		t.Fatal("saturday must be a day off by default")
	}

	_, err = svc.GetOwner(ctx, "missing")
	expectStatus(t, err, 404)

	_, err = svc.ListEvents(ctx, "missing")
	expectStatus(t, err, 404)

	_, err = svc.GetEvent(ctx, "1", "missing")
	expectStatus(t, err, 404)

	_, err = svc.ListBookings(ctx, "missing")
	expectStatus(t, err, 404)
}

func TestUpdateSchedule(t *testing.T) {
	svc, _ := newTestEnv(t, testNow)
	ctx := context.Background()

	sch := testOwner().Schedule
	sch.Monday.Start = "08:00"
	sch.Monday.End = "16:00"
	sch.Saturday.IsWorking = true

	owner, err := svc.UpdateSchedule(ctx, "1", sch)
	if err != nil {
		t.Fatalf("update schedule: %v", err)
	}
	if owner.Schedule.Monday.Start != "08:00" || owner.Schedule.Monday.End != "16:00" {
		t.Fatalf("schedule not updated: %+v", owner.Schedule.Monday)
	}
	if !owner.Schedule.Saturday.IsWorking {
		t.Fatal("saturday must be working after update")
	}

	bad := testOwner().Schedule
	bad.Monday.Start = "12:00"
	bad.Monday.End = "09:00"
	_, err = svc.UpdateSchedule(ctx, "1", bad)
	expectStatus(t, err, 400)

	bad2 := testOwner().Schedule
	bad2.Tuesday.Start = "25:00"
	_, err = svc.UpdateSchedule(ctx, "1", bad2)
	expectStatus(t, err, 400)
}

func TestCreateBookingAndList(t *testing.T) {
	svc, _ := newTestEnv(t, testNow)
	ctx := context.Background()
	ev := createEvent(t, svc, 60)

	booking, err := svc.CreateBooking(ctx, "1", ev.ID, model.BookingCreate{
		Name: "Иван", Email: "ivan@example.com",
		StartAt: utcDate(2026, 8, 10, 10, 0),
	})
	if err != nil {
		t.Fatalf("create booking: %v", err)
	}
	if booking.Date != "2026-08-10" {
		t.Fatalf("unexpected date: %s", booking.Date)
	}
	if !booking.EndAt.Equal(utcDate(2026, 8, 10, 11, 0)) {
		t.Fatalf("unexpected endAt: %s", booking.EndAt)
	}
	if booking.StartAt.Location() != time.UTC || booking.EndAt.Location() != time.UTC {
		t.Fatal("booking times must be UTC")
	}

	bookings, err := svc.ListBookings(ctx, "1")
	if err != nil {
		t.Fatalf("list bookings: %v", err)
	}
	if len(bookings) != 1 || bookings[0].ID != booking.ID {
		t.Fatalf("unexpected bookings: %+v", bookings)
	}
}

// Конкурентные бронирования одного слота: ровно одно успешно, остальные 409.
func TestCreateBookingConcurrent(t *testing.T) {
	svc, _ := newTestEnv(t, testNow)
	ctx := context.Background()
	ev := createEvent(t, svc, 60)
	start := utcDate(2026, 8, 10, 10, 0)

	const workers = 10
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		success  int
		conflict int
	)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := svc.CreateBooking(ctx, "1", ev.ID, model.BookingCreate{
				Name:    fmt.Sprintf("Гость %d", i),
				Email:   fmt.Sprintf("guest%d@example.com", i),
				StartAt: start,
			})
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				success++
				return
			}
			var ae *APIError
			if errors.As(err, &ae) && ae.Status == 409 {
				conflict++
				return
			}
			t.Errorf("unexpected error: %v", err)
		}(i)
	}
	wg.Wait()

	if success != 1 {
		t.Fatalf("expected exactly 1 success, got %d", success)
	}
	if conflict != workers-1 {
		t.Fatalf("expected %d conflicts, got %d", workers-1, conflict)
	}
}
