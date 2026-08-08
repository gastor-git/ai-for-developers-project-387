const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? ''

export interface DaySchedule {
  isWorking: boolean
  start: string
  end: string
}

export interface Schedule {
  monday: DaySchedule
  tuesday: DaySchedule
  wednesday: DaySchedule
  thursday: DaySchedule
  friday: DaySchedule
  saturday: DaySchedule
  sunday: DaySchedule
}

export interface Owner {
  id: string
  name: string
  schedule: Schedule
}

export interface Event {
  id: string
  ownerId: string
  title: string
  description: string
  durationMinutes: number
}

export interface EventCreate {
  title: string
  description: string
  durationMinutes: number
}

export interface Booking {
  id: string
  eventId: string
  date: string
  startAt: string
  endAt: string
  guestName: string
  guestEmail: string
  createdAt: string
}

export interface BookingCreate {
  name: string
  email: string
  startAt: string
}

export interface Slot {
  startAt: string
  endAt: string
}

export interface ApiErrorBody {
  code: string
  message: string
}

export class ApiError extends Error {
  status: number
  body: ApiErrorBody

  constructor(status: number, body: ApiErrorBody) {
    super(body.message)
    this.name = 'ApiError'
    this.status = status
    this.body = body
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let res: Response
  try {
    res = await fetch(`${API_BASE_URL}${path}`, {
      ...init,
      headers: {
        'Content-Type': 'application/json',
        ...init?.headers,
      },
    })
  } catch {
    throw new ApiError(0, {
      code: 'NETWORK_ERROR',
      message: 'Не удалось установить соединение с сервером',
    })
  }

  if (!res.ok) {
    let body: ApiErrorBody = {
      code: 'UNKNOWN',
      message: `Ошибка сервера (HTTP ${res.status})`,
    }
    try {
      const data = (await res.json()) as { error?: ApiErrorBody }
      if (data?.error) {
        body = data.error
      }
    } catch {
      // keep fallback body
    }
    throw new ApiError(res.status, body)
  }

  return res.json() as Promise<T>
}

export function getOwner(ownerId: string): Promise<Owner> {
  return request(`/api/owners/${encodeURIComponent(ownerId)}`)
}

export function updateSchedule(ownerId: string, schedule: Schedule): Promise<Owner> {
  return request(`/api/owners/${encodeURIComponent(ownerId)}/schedule`, {
    method: 'PATCH',
    body: JSON.stringify(schedule),
  })
}

export function listEvents(ownerId: string): Promise<Event[]> {
  return request(`/api/owners/${encodeURIComponent(ownerId)}/events`)
}

export function createEvent(ownerId: string, data: EventCreate): Promise<Event> {
  return request(`/api/owners/${encodeURIComponent(ownerId)}/events`, {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function getEvent(ownerId: string, eventId: string): Promise<Event> {
  return request(
    `/api/owners/${encodeURIComponent(ownerId)}/events/${encodeURIComponent(eventId)}`,
  )
}

export function getSlots(ownerId: string, eventId: string, date: string): Promise<Slot[]> {
  return request(
    `/api/owners/${encodeURIComponent(ownerId)}/events/${encodeURIComponent(eventId)}/slots?date=${encodeURIComponent(date)}`,
  )
}

export function createBooking(
  ownerId: string,
  eventId: string,
  data: BookingCreate,
): Promise<Booking> {
  return request(
    `/api/owners/${encodeURIComponent(ownerId)}/events/${encodeURIComponent(eventId)}/bookings`,
    {
      method: 'POST',
      body: JSON.stringify(data),
    },
  )
}

export function listBookings(ownerId: string): Promise<Booking[]> {
  return request(`/api/owners/${encodeURIComponent(ownerId)}/bookings`)
}

export function errorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    return err.body.message
  }
  if (err instanceof Error) {
    return err.message
  }
  return 'Неизвестная ошибка'
}

export const EMAIL_PATTERN = /^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$/
