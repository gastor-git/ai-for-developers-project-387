import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { addDays, format, startOfDay } from 'date-fns'
import {
  getEvent,
  getSlots,
  createBooking,
  errorMessage,
  EMAIL_PATTERN,
  type Event,
  type Slot,
} from '@/lib/api'
import { formatTime, formatDateTime } from '@/lib/format'
import { Calendar } from '@/components/ui/calendar'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { ApiErrorAlert } from '@/components/ApiErrorAlert'
import {
  ArrowLeft,
  CalendarCheck2,
  CalendarDays,
  CheckCircle2,
  Clock,
  Loader2,
  PartyPopper,
  UserRound,
} from 'lucide-react'

export function EventPage() {
  const { ownerId, eventId } = useParams<{ ownerId: string; eventId: string }>()

  const [event, setEvent] = useState<Event | null>(null)
  const [eventError, setEventError] = useState<string | null>(null)

  const windowStart = useMemo(() => startOfDay(new Date()), [])
  const windowEnd = useMemo(() => addDays(windowStart, 13), [windowStart])

  const [selectedDate, setSelectedDate] = useState<Date | undefined>(undefined)
  const [slots, setSlots] = useState<Slot[]>([])
  const [slotsLoading, setSlotsLoading] = useState(false)
  const [slotsError, setSlotsError] = useState<string | null>(null)

  const [selectedSlot, setSelectedSlot] = useState<Slot | null>(null)
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [formError, setFormError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [success, setSuccess] = useState<string | null>(null)

  useEffect(() => {
    if (!ownerId || !eventId) return
    let cancelled = false
    setEventError(null)
    getEvent(ownerId, eventId)
      .then((data) => {
        if (!cancelled) setEvent(data)
      })
      .catch((err) => {
        if (!cancelled) setEventError(errorMessage(err))
      })
    return () => {
      cancelled = true
    }
  }, [ownerId, eventId])

  const loadSlots = useCallback(
    async (date: Date) => {
      if (!ownerId || !eventId) return
      setSlotsLoading(true)
      setSlotsError(null)
      setSlots([])
      setSelectedSlot(null)
      try {
        const data = await getSlots(ownerId, eventId, format(date, 'yyyy-MM-dd'))
        setSlots(data)
      } catch (err) {
        setSlotsError(errorMessage(err))
      } finally {
        setSlotsLoading(false)
      }
    },
    [ownerId, eventId],
  )

  const handleDateSelect = (date: Date | undefined) => {
    setSelectedDate(date)
    if (date) {
      void loadSlots(date)
    } else {
      setSlots([])
      setSelectedSlot(null)
    }
  }

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setFormError(null)
    setSuccess(null)

    const trimmedName = name.trim()
    const trimmedEmail = email.trim()
    if (!trimmedName) {
      setFormError('Укажите ваше имя.')
      return
    }
    if (!trimmedEmail) {
      setFormError('Укажите ваш email.')
      return
    }
    if (!EMAIL_PATTERN.test(trimmedEmail)) {
      setFormError('Укажите корректный email, например name@example.com.')
      return
    }
    if (!selectedSlot) {
      setFormError('Выберите слот.')
      return
    }

    if (!ownerId || !eventId) return
    setSubmitting(true)
    try {
      const booking = await createBooking(ownerId, eventId, {
        name: trimmedName,
        email: trimmedEmail,
        startAt: selectedSlot.startAt,
      })
      setSuccess(
        `Бронирование подтверждено на ${formatDateTime(booking.startAt)}. Спасибо, ${booking.guestName}!`,
      )
      setName('')
      setEmail('')
      setSelectedSlot(null)
    } catch (err) {
      setFormError(errorMessage(err))
    } finally {
      setSubmitting(false)
    }
  }

  if (eventError) {
    return (
      <div className="mx-auto max-w-xl px-6 py-10">
        <ApiErrorAlert message={eventError} />
        <div className="mt-4">
          <Button asChild variant="outline" size="sm">
            <Link to="/">На главную</Link>
          </Button>
        </div>
      </div>
    )
  }

  if (!event) {
    return (
      <div className="flex items-center justify-center gap-2 py-16 text-muted-foreground">
        <Loader2 className="size-5 animate-spin" />
        Загрузка…
      </div>
    )
  }

  return (
    <div className="min-h-svh">
      <header className="border-b">
        <div className="mx-auto flex max-w-4xl items-center justify-between px-6 py-4">
          <Button asChild variant="ghost" size="sm">
            <Link to={`/owners/${ownerId}`}>
              <ArrowLeft className="size-4" />
              Календарь владельца
            </Link>
          </Button>
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <UserRound className="size-4" />
            Бронирование без регистрации
          </div>
        </div>
      </header>

      <main className="mx-auto grid max-w-4xl gap-6 px-6 py-8 lg:grid-cols-2">
        <div className="flex flex-col gap-6">
          <Card>
            <CardHeader>
              <CardTitle className="text-2xl">{event.title}</CardTitle>
              {event.description && (
                <CardDescription>{event.description}</CardDescription>
              )}
              <div className="flex items-center gap-2 pt-1 text-sm text-muted-foreground">
                <Clock className="size-4" />
                Длительность: {event.durationMinutes} мин
              </div>
            </CardHeader>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-lg">
                <CalendarDays className="size-5" />
                Выберите дату
              </CardTitle>
              <CardDescription>
                Доступны даты в ближайшие 14 дней.
              </CardDescription>
            </CardHeader>
            <CardContent className="flex justify-center">
              <Calendar
                mode="single"
                selected={selectedDate}
                onSelect={handleDateSelect}
                disabled={[
                  { before: windowStart },
                  { after: windowEnd },
                ]}
                defaultMonth={windowStart}
              />
            </CardContent>
          </Card>
        </div>

        <div className="flex flex-col gap-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-lg">
                <CalendarCheck2 className="size-5" />
                Свободные слоты
              </CardTitle>
              <CardDescription>
                {selectedDate
                  ? `Дата: ${selectedDate.toLocaleDateString('ru-RU')}`
                  : 'Выберите дату в календаре.'}
              </CardDescription>
            </CardHeader>
            <CardContent>
              {!selectedDate ? (
                <p className="py-6 text-center text-sm text-muted-foreground">
                  Начните с выбора даты.
                </p>
              ) : slotsLoading ? (
                <div className="flex items-center justify-center gap-2 py-6 text-muted-foreground">
                  <Loader2 className="size-4 animate-spin" />
                  Загрузка слотов…
                </div>
              ) : slotsError ? (
                <ApiErrorAlert message={slotsError} />
              ) : slots.length === 0 ? (
                <p className="py-6 text-center text-sm text-muted-foreground">
                  На эту дату нет свободных слотов.
                </p>
              ) : (
                <ul className="grid grid-cols-2 gap-2 sm:grid-cols-3">
                  {slots.map((slot) => {
                    const isSelected = selectedSlot?.startAt === slot.startAt
                    return (
                      <li key={slot.startAt}>
                        <Button
                          type="button"
                          variant={isSelected ? 'default' : 'outline'}
                          className="w-full"
                          onClick={() => {
                            setSelectedSlot(slot)
                            setFormError(null)
                          }}
                        >
                          {formatTime(slot.startAt)}
                        </Button>
                      </li>
                    )
                  })}
                </ul>
              )}
            </CardContent>
          </Card>

          {success && (
            <Card className="border-primary">
              <CardHeader>
                <div className="flex items-start gap-3">
                  <CheckCircle2 className="mt-0.5 size-6 shrink-0 text-primary" />
                  <div>
                    <CardTitle className="flex items-center gap-2 text-lg">
                      <PartyPopper className="size-5" />
                      Готово!
                    </CardTitle>
                    <CardDescription className="mt-1">
                      {success}
                    </CardDescription>
                  </div>
                </div>
              </CardHeader>
            </Card>
          )}

          {!success && (
            <Card>
              <CardHeader>
                <CardTitle className="text-lg">Ваши данные</CardTitle>
                <CardDescription>
                  {selectedSlot
                    ? `Слот ${formatTime(selectedSlot.startAt)} — ${formatTime(selectedSlot.endAt)}`
                    : 'Сначала выберите свободный слот.'}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <form className="grid gap-4" onSubmit={handleSubmit}>
                  <div className="grid gap-2">
                    <Label htmlFor="guest-name">Имя</Label>
                    <Input
                      id="guest-name"
                      value={name}
                      onChange={(e) => setName(e.target.value)}
                      placeholder="Иван Иванов"
                      autoComplete="name"
                    />
                  </div>
                  <div className="grid gap-2">
                    <Label htmlFor="guest-email">Email</Label>
                    <Input
                      id="guest-email"
                      type="email"
                      value={email}
                      onChange={(e) => setEmail(e.target.value)}
                      placeholder="ivan@example.com"
                      autoComplete="email"
                    />
                  </div>
                  {formError && <ApiErrorAlert message={formError} />}
                  <Button type="submit" disabled={submitting || !selectedSlot}>
                    {submitting && <Loader2 className="size-4 animate-spin" />}
                    Забронировать
                  </Button>
                </form>
              </CardContent>
            </Card>
          )}
        </div>
      </main>
    </div>
  )
}
