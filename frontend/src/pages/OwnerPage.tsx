import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  getOwner,
  listEvents,
  createEvent,
  errorMessage,
  type Owner,
  type Event,
} from '@/lib/api'
import { eventLink } from '@/lib/format'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { ApiErrorAlert } from '@/components/ApiErrorAlert'
import {
  ArrowLeft,
  CalendarDays,
  CalendarPlus,
  Check,
  Clock,
  Copy,
  Link2,
  ListChecks,
  Loader2,
  UserRound,
} from 'lucide-react'

const DAYS_RU = [
  'Понедельник',
  'Вторник',
  'Среда',
  'Четверг',
  'Пятница',
  'Суббота',
  'Воскресенье',
] as const

const SCHEDULE_KEYS = [
  'monday',
  'tuesday',
  'wednesday',
  'thursday',
  'friday',
  'saturday',
  'sunday',
] as const

export function OwnerPage() {
  const { ownerId } = useParams<{ ownerId: string }>()

  const [owner, setOwner] = useState<Owner | null>(null)
  const [events, setEvents] = useState<Event[]>([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)

  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [duration, setDuration] = useState('30')
  const [formError, setFormError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const [copiedId, setCopiedId] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!ownerId) return
    setLoading(true)
    setLoadError(null)
    try {
      const [ownerData, eventsData] = await Promise.all([
        getOwner(ownerId),
        listEvents(ownerId),
      ])
      setOwner(ownerData)
      setEvents(eventsData)
    } catch (err) {
      setLoadError(errorMessage(err))
    } finally {
      setLoading(false)
    }
  }, [ownerId])

  useEffect(() => {
    load()
  }, [load])

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setFormError(null)

    const trimmedTitle = title.trim()
    if (!trimmedTitle) {
      setFormError('Укажите название события.')
      return
    }
    const durationMinutes = Number(duration)
    if (
      !Number.isInteger(durationMinutes) ||
      durationMinutes < 15 ||
      durationMinutes > 480
    ) {
      setFormError('Длительность должна быть целым числом от 15 до 480 минут.')
      return
    }

    if (!ownerId) return
    setSubmitting(true)
    try {
      await createEvent(ownerId, {
        title: trimmedTitle,
        description: description.trim(),
        durationMinutes,
      })
      setTitle('')
      setDescription('')
      setDuration('30')
      await load()
    } catch (err) {
      setFormError(errorMessage(err))
    } finally {
      setSubmitting(false)
    }
  }

  const copyLink = async (event: Event) => {
    const link = eventLink(ownerId ?? '', event.id)
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(link)
      } else {
        const textarea = document.createElement('textarea')
        textarea.value = link
        textarea.style.position = 'fixed'
        textarea.style.opacity = '0'
        document.body.appendChild(textarea)
        textarea.select()
        document.execCommand('copy')
        document.body.removeChild(textarea)
      }
      setCopiedId(event.id)
      setTimeout(() => setCopiedId(null), 2000)
    } catch {
      setFormError('Не удалось скопировать ссылку.')
    }
  }

  return (
    <div className="min-h-svh">
      <header className="border-b">
        <div className="mx-auto flex max-w-3xl items-center justify-between px-6 py-4">
          <Button asChild variant="ghost" size="sm">
            <Link to="/">
              <ArrowLeft className="size-4" />
              На главную
            </Link>
          </Button>
          <Button asChild variant="outline" size="sm">
            <Link to={`/owners/${ownerId}/bookings`}>
              <ListChecks className="size-4" />
              Бронирования
            </Link>
          </Button>
        </div>
      </header>

      <main className="mx-auto flex max-w-3xl flex-col gap-6 px-6 py-8">
        {loading && (
          <div className="flex items-center justify-center gap-2 py-16 text-muted-foreground">
            <Loader2 className="size-5 animate-spin" />
            Загрузка…
          </div>
        )}

        {!loading && loadError && <ApiErrorAlert message={loadError} />}

        {!loading && owner && (
          <>
            <Card>
              <CardHeader>
                <div className="flex items-center gap-3">
                  <div className="flex size-12 items-center justify-center rounded-full bg-primary text-primary-foreground">
                    <UserRound className="size-6" />
                  </div>
                  <div>
                    <CardTitle className="text-2xl">{owner.name}</CardTitle>
                    <CardDescription>
                      Профиль владельца календаря · id: {owner.id}
                    </CardDescription>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="grid gap-3">
                <div className="flex items-center gap-2 text-sm font-medium">
                  <CalendarDays className="size-4 text-muted-foreground" />
                  График работы
                </div>
                <ul className="grid gap-1.5">
                  {SCHEDULE_KEYS.map((key, index) => {
                    const day = owner.schedule[key]
                    return (
                      <li
                        key={key}
                        className="flex items-center justify-between text-sm"
                      >
                        <span className="text-muted-foreground">
                          {DAYS_RU[index]}
                        </span>
                        {day.isWorking ? (
                          <Badge variant="secondary">
                            <Clock className="size-3" />
                            {day.start}–{day.end}
                          </Badge>
                        ) : (
                          <Badge variant="outline">Выходной</Badge>
                        )}
                      </li>
                    )
                  })}
                </ul>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-lg">
                  <CalendarPlus className="size-5" />
                  Новое событие
                </CardTitle>
                <CardDescription>
                  Создайте тип записи, который сможет забронировать гость.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <form className="grid gap-4" onSubmit={handleSubmit}>
                  <div className="grid gap-2">
                    <Label htmlFor="event-title">Название</Label>
                    <Input
                      id="event-title"
                      value={title}
                      onChange={(e) => setTitle(e.target.value)}
                      placeholder="Например: Консультация"
                    />
                  </div>
                  <div className="grid gap-2">
                    <Label htmlFor="event-description">Описание</Label>
                    <Textarea
                      id="event-description"
                      value={description}
                      onChange={(e) => setDescription(e.target.value)}
                      placeholder="Кратко о том, что будет на встрече"
                      rows={3}
                    />
                  </div>
                  <div className="grid max-w-xs gap-2">
                    <Label htmlFor="event-duration">Длительность, минут</Label>
                    <Input
                      id="event-duration"
                      type="number"
                      min={15}
                      max={480}
                      value={duration}
                      onChange={(e) => setDuration(e.target.value)}
                    />
                  </div>
                  {formError && <ApiErrorAlert message={formError} />}
                  <div>
                    <Button type="submit" disabled={submitting}>
                      {submitting && (
                        <Loader2 className="size-4 animate-spin" />
                      )}
                      Создать событие
                    </Button>
                  </div>
                </form>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="text-lg">События</CardTitle>
                <CardDescription>
                  Ссылки на события можно скопировать и отправить гостям.
                </CardDescription>
              </CardHeader>
              <CardContent>
                {events.length === 0 ? (
                  <p className="py-6 text-center text-sm text-muted-foreground">
                    Событий пока нет. Создайте первое событие выше.
                  </p>
                ) : (
                  <ul className="flex flex-col">
                    {events.map((event, index) => (
                      <li key={event.id}>
                        {index > 0 && <Separator />}
                        <div className="flex items-start justify-between gap-4 py-4">
                          <div className="min-w-0">
                            <div className="flex flex-wrap items-center gap-2">
                              <Link
                                to={`/owners/${ownerId}/events/${event.id}`}
                                className="font-medium hover:underline"
                              >
                                {event.title}
                              </Link>
                              <Badge variant="secondary">
                                <Clock className="size-3" />
                                {event.durationMinutes} мин
                              </Badge>
                            </div>
                            {event.description && (
                              <p className="mt-1 line-clamp-2 text-sm text-muted-foreground">
                                {event.description}
                              </p>
                            )}
                          </div>
                          <div className="flex shrink-0 gap-2">
                            <Button
                              asChild
                              variant="outline"
                              size="sm"
                            >
                              <Link
                                to={`/owners/${ownerId}/events/${event.id}`}
                              >
                                <Link2 className="size-4" />
                                Страница
                              </Link>
                            </Button>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => copyLink(event)}
                            >
                              {copiedId === event.id ? (
                                <Check className="size-4" />
                              ) : (
                                <Copy className="size-4" />
                              )}
                              {copiedId === event.id ? 'Скопировано' : 'Копировать'}
                            </Button>
                          </div>
                        </div>
                      </li>
                    ))}
                  </ul>
                )}
              </CardContent>
            </Card>
          </>
        )}
      </main>
    </div>
  )
}
