package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"booking-calendar/backend/internal/httpapi"
	"booking-calendar/backend/internal/model"
	"booking-calendar/backend/internal/service"
	"booking-calendar/backend/internal/store"
)

const defaultOwnerID = "1"

func defaultOwner() model.Owner {
	return model.Owner{
		ID:   defaultOwnerID,
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

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func loadLocation() *time.Location {
	name := os.Getenv("APP_TIMEZONE")
	if name == "" {
		return time.Local
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		log.Fatalf("некорректная APP_TIMEZONE %q: %v", name, err)
	}
	return loc
}

func main() {
	ctx := context.Background()

	dbPath := envOr("DB_PATH", "booking.db")
	loc := loadLocation()

	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	if err := st.Migrate(ctx); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if err := st.SeedOwner(ctx, defaultOwner()); err != nil {
		log.Fatalf("seed owner: %v", err)
	}

	svc := service.New(st, loc, time.Now)
	handler := httpapi.NewHandler(svc)

	addr := envOr("ADDR", ":8080")
	log.Printf("booking backend listening on %s (db=%s, tz=%s)", addr, dbPath, loc)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
