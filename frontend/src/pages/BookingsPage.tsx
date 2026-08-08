import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  getOwner,
  listBookings,
  listEvents,
  errorMessage,
  type Booking,
  type Event,
} from '@/lib/api'
import { formatDateTime } from '@/lib/format'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { ApiErrorAlert } from '@/components/ApiErrorAlert'
import { ArrowLeft, CalendarCheck2, Loader2, UserRound } from 'lucide-react'

export function BookingsPage() {
  const { ownerId } = useParams<{ ownerId: string }>()

  const [bookings, setBookings] = useState<Booking[]>([])
  const [events, setEvents] = useState<Event[]>([])
  const [ownerName, setOwnerName] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!ownerId) return
    setLoading(true)
    setLoadError(null)
    try {
      const [bookingsData, eventsData, ownerData] = await Promise.all([
        listBookings(ownerId),
        listEvents(ownerId),
        getOwner(ownerId),
      ])
      setBookings(bookingsData)
      setEvents(eventsData)
      setOwnerName(ownerData.name)
    } catch (err) {
      setLoadError(errorMessage(err))
    } finally {
      setLoading(false)
    }
  }, [ownerId])

  useEffect(() => {
    load()
  }, [load])

  const eventById = new Map(events.map((event) => [event.id, event]))

  return (
    <div className="min-h-svh">
      <header className="border-b">
        <div className="mx-auto flex max-w-3xl items-center justify-between px-6 py-4">
          <Button asChild variant="ghost" size="sm">
            <Link to={`/owners/${ownerId}`}>
              <ArrowLeft className="size-4" />
              Назад к календарю
            </Link>
          </Button>
          {ownerName && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <UserRound className="size-4" />
              {ownerName}
            </div>
          )}
        </div>
      </header>

      <main className="mx-auto max-w-3xl px-6 py-8">
        {loading && (
          <div className="flex items-center justify-center gap-2 py-16 text-muted-foreground">
            <Loader2 className="size-5 animate-spin" />
            Загрузка…
          </div>
        )}

        {!loading && loadError && <ApiErrorAlert message={loadError} />}

        {!loading && !loadError && (
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-xl">
                <CalendarCheck2 className="size-5" />
                Бронирования
              </CardTitle>
              <CardDescription>
                Все записи гостей на ваши события.
              </CardDescription>
            </CardHeader>
            <CardContent>
              {bookings.length === 0 ? (
                <p className="py-6 text-center text-sm text-muted-foreground">
                  Бронирований пока нет.
                </p>
              ) : (
                <ul className="flex flex-col">
                  {bookings.map((booking, index) => {
                    const event = eventById.get(booking.eventId)
                    return (
                      <li key={booking.id}>
                        {index > 0 && <Separator />}
                        <div className="flex items-start justify-between gap-4 py-4">
                          <div className="min-w-0">
                            <div className="flex flex-wrap items-center gap-2">
                              <span className="font-medium">
                                {formatDateTime(booking.startAt)}
                              </span>
                              <Badge variant="secondary">
                                {event?.title ?? booking.eventId}
                              </Badge>
                            </div>
                            <p className="mt-1 text-sm text-muted-foreground">
                              {booking.guestName} · {booking.guestEmail}
                            </p>
                          </div>
                          <span className="shrink-0 text-sm text-muted-foreground">
                            {booking.date}
                          </span>
                        </div>
                      </li>
                    )
                  })}
                </ul>
              )}
            </CardContent>
          </Card>
        )}
      </main>
    </div>
  )
}
