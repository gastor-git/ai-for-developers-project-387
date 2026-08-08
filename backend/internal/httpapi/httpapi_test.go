package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"booking-calendar/backend/internal/httpapi"
	"booking-calendar/backend/internal/model"
	"booking-calendar/backend/internal/service"
	"booking-calendar/backend/internal/store"
)

var (
	testLoc = time.UTC
	// Понедельник, 2026-08-10 08:00 UTC. График: Пн–Пт 09:00–18:00.
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

func newTestServer(t *testing.T) *httptest.Server {
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
	svc := service.New(st, testLoc, func() time.Time { return testNow })
	ts := httptest.NewServer(httpapi.NewHandler(svc))
	t.Cleanup(ts.Close)
	return ts
}

func do(t *testing.T, method, url string, body any) (int, []byte) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, data
}

func decode(t *testing.T, data []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("decode %s: %v", string(data), err)
	}
}

func expectStatus(t *testing.T, got, want int, data []byte) {
	t.Helper()
	if got != want {
		t.Fatalf("expected status %d, got %d (body: %s)", want, got, data)
	}
}

func createEvent(t *testing.T, base string) model.Event {
	t.Helper()
	status, data := do(t, http.MethodPost, base+"/api/owners/1/events", map[string]any{
		"title": "Консультация", "description": "Описание", "durationMinutes": 60,
	})
	expectStatus(t, status, 200, data)
	var ev model.Event
	decode(t, data, &ev)
	return ev
}

func TestHealth(t *testing.T) {
	ts := newTestServer(t)
	status, data := do(t, http.MethodGet, ts.URL+"/api/health", nil)
	expectStatus(t, status, 200, data)
}

func TestGetOwner(t *testing.T) {
	ts := newTestServer(t)
	status, data := do(t, http.MethodGet, ts.URL+"/api/owners/1", nil)
	expectStatus(t, status, 200, data)

	var owner model.Owner
	decode(t, data, &owner)
	if owner.ID != "1" || owner.Name == "" {
		t.Fatalf("unexpected owner: %+v", owner)
	}
	if !owner.Schedule.Monday.IsWorking || owner.Schedule.Saturday.IsWorking {
		t.Fatalf("unexpected default schedule: %+v", owner.Schedule)
	}

	status, data = do(t, http.MethodGet, ts.URL+"/api/owners/missing", nil)
	expectStatus(t, status, 404, data)
	assertErrorCode(t, data, "NOT_FOUND")
}

func TestUpdateSchedule(t *testing.T) {
	ts := newTestServer(t)
	sch := testOwner().Schedule
	sch.Monday.Start = "08:00"
	sch.Saturday.IsWorking = true

	status, data := do(t, http.MethodPatch, ts.URL+"/api/owners/1/schedule", sch)
	expectStatus(t, status, 200, data)
	var owner model.Owner
	decode(t, data, &owner)
	if owner.Schedule.Monday.Start != "08:00" || !owner.Schedule.Saturday.IsWorking {
		t.Fatalf("schedule not updated: %+v", owner.Schedule)
	}

	bad := testOwner().Schedule
	bad.Tuesday.Start = "09:00"
	bad.Tuesday.End = "08:00"
	status, data = do(t, http.MethodPatch, ts.URL+"/api/owners/1/schedule", bad)
	expectStatus(t, status, 400, data)
	assertErrorCode(t, data, "BAD_REQUEST")
}

func TestEventFlow(t *testing.T) {
	ts := newTestServer(t)
	ev := createEvent(t, ts.URL)

	if ev.ID == "" || ev.OwnerID != "1" || ev.DurationMinutes != 60 {
		t.Fatalf("unexpected event: %+v", ev)
	}

	status, data := do(t, http.MethodGet, ts.URL+"/api/owners/1/events/"+ev.ID, nil)
	expectStatus(t, status, 200, data)
	var got model.Event
	decode(t, data, &got)
	if got.ID != ev.ID {
		t.Fatalf("unexpected event: %+v", got)
	}

	status, data = do(t, http.MethodGet, ts.URL+"/api/owners/1/events", nil)
	expectStatus(t, status, 200, data)
	var events []model.Event
	decode(t, data, &events)
	if len(events) != 1 || events[0].ID != ev.ID {
		t.Fatalf("unexpected events: %+v", events)
	}
}

func TestCreateEventValidation(t *testing.T) {
	ts := newTestServer(t)

	status, data := do(t, http.MethodPost, ts.URL+"/api/owners/1/events", map[string]any{
		"title": "X", "durationMinutes": 10,
	})
	expectStatus(t, status, 400, data)
	assertErrorCode(t, data, "BAD_REQUEST")

	status, data = do(t, http.MethodPost, ts.URL+"/api/owners/missing/events", map[string]any{
		"title": "X", "durationMinutes": 30,
	})
	expectStatus(t, status, 404, data)
	assertErrorCode(t, data, "NOT_FOUND")
}

func TestSlots(t *testing.T) {
	ts := newTestServer(t)
	ev := createEvent(t, ts.URL)

	status, data := do(t, http.MethodGet, ts.URL+"/api/owners/1/events/"+ev.ID+"/slots?date=2026-08-10", nil)
	expectStatus(t, status, 200, data)
	var slots []model.Slot
	decode(t, data, &slots)
	if len(slots) == 0 {
		t.Fatal("expected slots")
	}
	if slots[0].StartAt.Location() != time.UTC {
		t.Fatalf("slots must be UTC, got %s", slots[0].StartAt)
	}

	status, data = do(t, http.MethodGet, ts.URL+"/api/owners/1/events/"+ev.ID+"/slots?date=2026-08-15", nil)
	expectStatus(t, status, 200, data)
	decode(t, data, &slots)
	if len(slots) != 0 {
		t.Fatalf("expected no slots on weekend, got %d", len(slots))
	}

	status, data = do(t, http.MethodGet, ts.URL+"/api/owners/1/events/"+ev.ID+"/slots?date=not-a-date", nil)
	expectStatus(t, status, 400, data)
	assertErrorCode(t, data, "BAD_REQUEST")
}

// Интеграционный тест на 409: занятый слот отклоняется с телом ConflictError.
func TestBookingConflict409(t *testing.T) {
	ts := newTestServer(t)
	ev := createEvent(t, ts.URL)
	booking := map[string]any{
		"name": "Иван", "email": "ivan@example.com", "startAt": "2026-08-10T10:00:00Z",
	}

	status, data := do(t, http.MethodPost, ts.URL+"/api/owners/1/events/"+ev.ID+"/bookings", booking)
	expectStatus(t, status, 200, data)

	status, data = do(t, http.MethodPost, ts.URL+"/api/owners/1/events/"+ev.ID+"/bookings", booking)
	expectStatus(t, status, 409, data)
	assertErrorCode(t, data, "CONFLICT")

	var errBody struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	decode(t, data, &errBody)
	if errBody.Error.Message == "" {
		t.Fatal("expected human-readable message in 409 body")
	}
}

func TestBookingFlow(t *testing.T) {
	ts := newTestServer(t)
	ev := createEvent(t, ts.URL)

	status, data := do(t, http.MethodPost, ts.URL+"/api/owners/1/events/"+ev.ID+"/bookings", map[string]any{
		"name": "Иван", "email": "ivan@example.com", "startAt": "2026-08-10T10:00:00Z",
	})
	expectStatus(t, status, 200, data)
	var booking model.Booking
	decode(t, data, &booking)
	if booking.ID == "" || booking.EventID != ev.ID || booking.Date != "2026-08-10" {
		t.Fatalf("unexpected booking: %+v", booking)
	}
	if booking.StartAt.IsZero() || booking.EndAt.IsZero() || booking.CreatedAt.IsZero() {
		t.Fatalf("booking times missing: %+v", booking)
	}

	status, data = do(t, http.MethodGet, ts.URL+"/api/owners/1/bookings", nil)
	expectStatus(t, status, 200, data)
	var bookings []model.Booking
	decode(t, data, &bookings)
	if len(bookings) != 1 || bookings[0].ID != booking.ID || bookings[0].GuestEmail != "ivan@example.com" {
		t.Fatalf("unexpected bookings: %+v", bookings)
	}
}

func TestBookingValidation(t *testing.T) {
	ts := newTestServer(t)
	ev := createEvent(t, ts.URL)
	url := ts.URL + "/api/owners/1/events/" + ev.ID + "/bookings"

	cases := []struct {
		name string
		body map[string]any
	}{
		{"empty name", map[string]any{"name": "", "email": "a@b.ru", "startAt": "2026-08-10T10:00:00Z"}},
		{"bad email", map[string]any{"name": "Иван", "email": "nope", "startAt": "2026-08-10T10:00:00Z"}},
		{"past slot", map[string]any{"name": "Иван", "email": "a@b.ru", "startAt": "2026-08-10T07:00:00Z"}},
		{"weekend", map[string]any{"name": "Иван", "email": "a@b.ru", "startAt": "2026-08-15T10:00:00Z"}},
		{"malformed json", nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, data := do(t, http.MethodPost, url, tc.body)
			expectStatus(t, status, 400, data)
			assertErrorCode(t, data, "BAD_REQUEST")
		})
	}
}

func TestNotFound(t *testing.T) {
	ts := newTestServer(t)

	status, data := do(t, http.MethodGet, ts.URL+"/api/owners/missing/events", nil)
	expectStatus(t, status, 404, data)

	status, data = do(t, http.MethodGet, ts.URL+"/api/owners/1/events/missing", nil)
	expectStatus(t, status, 404, data)

	status, data = do(t, http.MethodGet, ts.URL+"/api/owners/1/events/missing/slots?date=2026-08-10", nil)
	expectStatus(t, status, 404, data)

	status, data = do(t, http.MethodGet, ts.URL+"/api/owners/missing/bookings", nil)
	expectStatus(t, status, 404, data)

	status, data = do(t, http.MethodPatch, ts.URL+"/api/owners/missing/schedule", testOwner().Schedule)
	expectStatus(t, status, 404, data)
}

func assertErrorCode(t *testing.T, data []byte, want string) {
	t.Helper()
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("decode error body %s: %v", string(data), err)
	}
	if body.Error.Code != want {
		t.Fatalf("expected error code %s, got %q (body: %s)", want, body.Error.Code, data)
	}
}
