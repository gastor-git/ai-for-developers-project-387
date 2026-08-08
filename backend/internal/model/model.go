package model

import "time"

type DaySchedule struct {
	IsWorking bool   `json:"isWorking"`
	Start     string `json:"start"`
	End       string `json:"end"`
}

type Schedule struct {
	Monday    DaySchedule `json:"monday"`
	Tuesday   DaySchedule `json:"tuesday"`
	Wednesday DaySchedule `json:"wednesday"`
	Thursday  DaySchedule `json:"thursday"`
	Friday    DaySchedule `json:"friday"`
	Saturday  DaySchedule `json:"saturday"`
	Sunday    DaySchedule `json:"sunday"`
}

type Owner struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Schedule Schedule `json:"schedule"`
}

type Event struct {
	ID              string `json:"id"`
	OwnerID         string `json:"ownerId"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	DurationMinutes int    `json:"durationMinutes"`
}

type EventCreate struct {
	Title           string `json:"title"`
	Description     string `json:"description"`
	DurationMinutes int    `json:"durationMinutes"`
}

type Booking struct {
	ID         string    `json:"id"`
	EventID    string    `json:"eventId"`
	Date       string    `json:"date"`
	StartAt    time.Time `json:"startAt"`
	EndAt      time.Time `json:"endAt"`
	GuestName  string    `json:"guestName"`
	GuestEmail string    `json:"guestEmail"`
	CreatedAt  time.Time `json:"createdAt"`
}

type BookingCreate struct {
	Name    string    `json:"name"`
	Email   string    `json:"email"`
	StartAt time.Time `json:"startAt"`
}

type Slot struct {
	StartAt time.Time `json:"startAt"`
	EndAt   time.Time `json:"endAt"`
}
