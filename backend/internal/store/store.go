package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"booking-calendar/backend/internal/model"
)

var ErrConflict = errors.New("slot already taken")

const (
	timeLayout = time.RFC3339
	driverName = "sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	dsn := fmt.Sprintf("%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)", path)
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *Store) Migrate(ctx context.Context) error {
	const schema = `
CREATE TABLE IF NOT EXISTS owners (
    id       TEXT PRIMARY KEY,
    name     TEXT NOT NULL,
    schedule TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS events (
    id               TEXT PRIMARY KEY,
    owner_id         TEXT NOT NULL REFERENCES owners(id) ON DELETE CASCADE,
    title            TEXT NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    duration_minutes INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS bookings (
    id          TEXT PRIMARY KEY,
    event_id    TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    date        TEXT NOT NULL,
    start_at    TEXT NOT NULL,
    end_at      TEXT NOT NULL,
    guest_name  TEXT NOT NULL,
    guest_email TEXT NOT NULL,
    created_at  TEXT NOT NULL,
    UNIQUE (date, start_at)
);

CREATE INDEX IF NOT EXISTS idx_events_owner_id ON events(owner_id);
CREATE INDEX IF NOT EXISTS idx_bookings_event_id ON bookings(event_id);
CREATE INDEX IF NOT EXISTS idx_bookings_date ON bookings(date);
`
	_, err := s.db.ExecContext(ctx, schema)
	return err
}

func (s *Store) SeedOwner(ctx context.Context, owner model.Owner) error {
	scheduleJSON, err := json.Marshal(owner.Schedule)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO owners (id, name, schedule) VALUES (?, ?, ?)
		 ON CONFLICT(id) DO NOTHING`,
		owner.ID, owner.Name, string(scheduleJSON),
	)
	return err
}

func (s *Store) GetOwner(ctx context.Context, id string) (*model.Owner, error) {
	var (
		owner        model.Owner
		scheduleJSON string
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, schedule FROM owners WHERE id = ?`, id,
	).Scan(&owner.ID, &owner.Name, &scheduleJSON)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(scheduleJSON), &owner.Schedule); err != nil {
		return nil, err
	}
	return &owner, nil
}

func (s *Store) UpdateOwnerSchedule(ctx context.Context, id string, schedule model.Schedule) error {
	scheduleJSON, err := json.Marshal(schedule)
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE owners SET schedule = ? WHERE id = ?`, string(scheduleJSON), id,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) ListEvents(ctx context.Context, ownerID string) ([]model.Event, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, owner_id, title, description, duration_minutes
		 FROM events WHERE owner_id = ? ORDER BY rowid`,
		ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]model.Event, 0)
	for rows.Next() {
		var e model.Event
		if err := rows.Scan(&e.ID, &e.OwnerID, &e.Title, &e.Description, &e.DurationMinutes); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (s *Store) GetEvent(ctx context.Context, ownerID, eventID string) (*model.Event, error) {
	var e model.Event
	err := s.db.QueryRowContext(ctx,
		`SELECT id, owner_id, title, description, duration_minutes
		 FROM events WHERE id = ? AND owner_id = ?`,
		eventID, ownerID,
	).Scan(&e.ID, &e.OwnerID, &e.Title, &e.Description, &e.DurationMinutes)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (s *Store) CreateEvent(ctx context.Context, ownerID string, ec model.EventCreate, id string) (*model.Event, error) {
	ev := model.Event{
		ID:              id,
		OwnerID:         ownerID,
		Title:           ec.Title,
		Description:     ec.Description,
		DurationMinutes: ec.DurationMinutes,
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO events (id, owner_id, title, description, duration_minutes)
		 VALUES (?, ?, ?, ?, ?)`,
		ev.ID, ev.OwnerID, ev.Title, ev.Description, ev.DurationMinutes,
	)
	if err != nil {
		return nil, err
	}
	return &ev, nil
}

func (s *Store) BookingsOnDate(ctx context.Context, date string) ([]model.Booking, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, event_id, date, start_at, end_at, guest_name, guest_email, created_at
		 FROM bookings WHERE date = ?`,
		date,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bookings := make([]model.Booking, 0)
	for rows.Next() {
		b, err := scanBooking(rows)
		if err != nil {
			return nil, err
		}
		bookings = append(bookings, b)
	}
	return bookings, rows.Err()
}

func (s *Store) ListBookings(ctx context.Context, ownerID string) ([]model.Booking, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT b.id, b.event_id, b.date, b.start_at, b.end_at, b.guest_name, b.guest_email, b.created_at
		 FROM bookings b
		 JOIN events e ON e.id = b.event_id
		 WHERE e.owner_id = ?
		 ORDER BY b.start_at`,
		ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bookings := make([]model.Booking, 0)
	for rows.Next() {
		b, err := scanBooking(rows)
		if err != nil {
			return nil, err
		}
		bookings = append(bookings, b)
	}
	return bookings, rows.Err()
}

func (s *Store) CreateBooking(ctx context.Context, b *model.Booking) error {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return err
	}
	rolledBack := true
	defer func() {
		if rolledBack {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()

	var count int
	err = conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM bookings WHERE date = ? AND start_at < ? AND end_at > ?`,
		b.Date, formatTime(b.EndAt), formatTime(b.StartAt),
	).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrConflict
	}

	_, err = conn.ExecContext(ctx,
		`INSERT INTO bookings (id, event_id, date, start_at, end_at, guest_name, guest_email, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		b.ID, b.EventID, b.Date,
		formatTime(b.StartAt), formatTime(b.EndAt),
		b.GuestName, b.GuestEmail, formatTime(b.CreatedAt),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}

	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return err
	}
	rolledBack = false
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanBooking(row rowScanner) (model.Booking, error) {
	var (
		b              model.Booking
		startAt, endAt string
		createdAt      string
	)
	err := row.Scan(
		&b.ID, &b.EventID, &b.Date,
		&startAt, &endAt,
		&b.GuestName, &b.GuestEmail, &createdAt,
	)
	if err != nil {
		return model.Booking{}, err
	}
	if b.StartAt, err = parseTime(startAt); err != nil {
		return model.Booking{}, err
	}
	if b.EndAt, err = parseTime(endAt); err != nil {
		return model.Booking{}, err
	}
	if b.CreatedAt, err = parseTime(createdAt); err != nil {
		return model.Booking{}, err
	}
	return b, nil
}

func formatTime(t time.Time) string {
	return t.UTC().Format(timeLayout)
}

func parseTime(v string) (time.Time, error) {
	return time.Parse(timeLayout, v)
}

func isUniqueViolation(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "constraint failed")
}
