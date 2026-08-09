import { expect, type Page } from '@playwright/test'

const DAY_KEY_LOCALE = 'en-US'

// Ключ для data-day календаря (react-day-picker отдаёт toLocaleDateString()).
export function dataDayKey(date: Date): string {
  return date.toLocaleDateString(DAY_KEY_LOCALE)
}

// «Следующий рабочий день» строго после сегодня: слоты генерируются только
// в будущем (startAt > now), а выходные в графике по умолчанию закрыты.
export function nextWorkingDay(now: Date): Date {
  const start = new Date(now)
  start.setHours(0, 0, 0, 0)
  for (let i = 1; i <= 13; i++) {
    const candidate = new Date(start)
    candidate.setDate(start.getDate() + i)
    const weekday = candidate.getDay()
    if (weekday !== 0 && weekday !== 6) {
      return candidate
    }
  }
  throw new Error('В окне записи 14 дней нет рабочего дня')
}

// Выбирает день в календаре страницы события; при необходимости
// перелистывает месяц вперёд (окно записи не длиннее 14 дней).
export async function selectWorkingDay(page: Page, day: Date): Promise<void> {
  const today = new Date()
  if (day.getMonth() !== today.getMonth()) {
    await page
      .getByRole('button', { name: 'Go to the Next Month' })
      .click()
  }
  const dayButton = page.locator(`[data-day="${dataDayKey(day)}"]`)
  await expect(dayButton).toBeVisible()
  await dayButton.click()
}

export async function selectFirstSlot(page: Page): Promise<void> {
  const firstSlot = page.locator('ul button').first()
  await expect(firstSlot).toBeVisible()
  await firstSlot.click()
}

export function eventUrl(eventId: string): string {
  return `/owners/1/events/${eventId}`
}
