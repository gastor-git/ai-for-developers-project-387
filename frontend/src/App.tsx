import { useState } from 'react'
import { BrowserRouter, Routes, Route, Link, useNavigate } from 'react-router-dom'
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
import { CalendarDays, CalendarHeart } from 'lucide-react'
import { OwnerPage } from '@/pages/OwnerPage'
import { EventPage } from '@/pages/EventPage'
import { BookingsPage } from '@/pages/BookingsPage'

function HomePage() {
  const navigate = useNavigate()
  const [ownerId, setOwnerId] = useState('1')

  return (
    <div className="flex min-h-svh items-center justify-center p-6">
      <Card className="w-full max-w-md">
        <CardHeader>
          <div className="mb-2 flex size-12 items-center justify-center rounded-xl bg-primary text-primary-foreground">
            <CalendarHeart className="size-6" />
          </div>
          <CardTitle className="text-2xl">Календарь бронирований</CardTitle>
          <CardDescription>
            Введите идентификатор владельца календаря, чтобы открыть его страницу.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form
            className="flex flex-col gap-4"
            onSubmit={(e) => {
              e.preventDefault()
              if (ownerId.trim()) {
                navigate(`/owners/${encodeURIComponent(ownerId.trim())}`)
              }
            }}
          >
            <div className="grid gap-2">
              <Label htmlFor="owner-id">ID владельца</Label>
              <Input
                id="owner-id"
                value={ownerId}
                onChange={(e) => setOwnerId(e.target.value)}
                placeholder="1"
              />
            </div>
            <Button type="submit">Открыть календарь</Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}

function NotFoundPage() {
  return (
    <div className="flex min-h-svh flex-col items-center justify-center gap-4 p-6 text-center">
      <CalendarDays className="size-12 text-muted-foreground" />
      <h1 className="text-2xl font-semibold">Страница не найдена</h1>
      <Button asChild>
        <Link to="/">На главную</Link>
      </Button>
    </div>
  )
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/owners/:ownerId" element={<OwnerPage />} />
        <Route path="/owners/:ownerId/events/:eventId" element={<EventPage />} />
        <Route path="/owners/:ownerId/bookings" element={<BookingsPage />} />
        <Route path="*" element={<NotFoundPage />} />
      </Routes>
    </BrowserRouter>
  )
}
