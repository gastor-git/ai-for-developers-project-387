import { test, expect } from '@playwright/test'
import {
  nextWorkingDay,
  selectFirstSlot,
  selectWorkingDay,
  eventUrl,
} from './utils'

const EVENT_TITLE = 'Консультация по проекту'
const EVENT_DESCRIPTION = 'Разбор архитектуры и плана работ'
const EVENT_DURATION = '30'
const GUEST_NAME = 'Иван Иванов'
const GUEST_EMAIL = 'ivan@example.com'

test.describe.serial('Сценарий бронирования', () => {
  let eventId = ''
  let bookingDate = new Date()

  test('1. Владелец создаёт событие через UI и видит его в списке', async ({
    page,
  }) => {
    await page.goto('/owners/1')
    await expect(page.getByText('Владелец календаря')).toBeVisible()

    await page.getByLabel('Название').fill(EVENT_TITLE)
    await page.getByLabel('Описание').fill(EVENT_DESCRIPTION)
    await page.getByLabel('Длительность, минут').fill(EVENT_DURATION)
    await page.getByRole('button', { name: 'Создать событие' }).click()

    const eventLink = page.getByRole('link', { name: EVENT_TITLE })
    await expect(eventLink).toBeVisible()
    const href = await eventLink.getAttribute('href')
    expect(href).toBeTruthy()
    eventId = href!.split('/').pop()!
    expect(eventId).not.toBe('')
  })

  test('2. Гость бронирует слот и получает подтверждение', async ({ page }) => {
    expect(eventId).not.toBe('')
    bookingDate = nextWorkingDay(new Date())

    await page.goto(eventUrl(eventId))
    await expect(page.getByText(EVENT_TITLE)).toBeVisible()

    await selectWorkingDay(page, bookingDate)
    await selectFirstSlot(page)

    await page.getByLabel('Имя').fill(GUEST_NAME)
    await page.getByLabel('Email').fill(GUEST_EMAIL)
    await page.getByRole('button', { name: 'Забронировать' }).click()

    await expect(page.getByText('Готово!')).toBeVisible()
    await expect(page.getByText(`Спасибо, ${GUEST_NAME}!`)).toBeVisible()
  })

  test('3. Владелец видит бронь на странице /bookings', async ({ page }) => {
    await page.goto('/owners/1/bookings')
    await expect(page.getByText(GUEST_EMAIL)).toBeVisible()
    await expect(page.getByText(GUEST_NAME)).toBeVisible()
  })

  test('4. Конфликт 409: бронь того же слота дважды отклоняется', async ({
    page,
  }) => {
    expect(eventId).not.toBe('')
    const secondPage = await page.context().newPage()

    // Обе страницы выбирают одну и ту же дату и слот, пока он свободен.
    for (const p of [page, secondPage]) {
      await p.goto(eventUrl(eventId))
      await selectWorkingDay(p, bookingDate)
      await selectFirstSlot(p)
      await p.getByLabel('Имя').fill(GUEST_NAME)
      await p.getByLabel('Email').fill(GUEST_EMAIL)
    }

    await page.getByRole('button', { name: 'Забронировать' }).click()
    await expect(page.getByText('Готово!')).toBeVisible()

    await secondPage
      .getByRole('button', { name: 'Забронировать' })
      .click()
    await expect(secondPage.getByText('Выбранный слот уже занят')).toBeVisible()

    await secondPage.close()
  })
})
