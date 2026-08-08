import { format } from 'date-fns'

export function formatDateTime(iso: string): string {
  return format(new Date(iso), 'dd.MM.yyyy, HH:mm')
}

export function formatTime(iso: string): string {
  return format(new Date(iso), 'HH:mm')
}

export function formatDate(iso: string): string {
  return format(new Date(iso), 'dd.MM.yyyy')
}

export function eventLink(ownerId: string, eventId: string): string {
  return `${window.location.origin}/owners/${encodeURIComponent(ownerId)}/events/${encodeURIComponent(eventId)}`
}
