import "@typespec/http";

using TypeSpec.Http;

namespace BookingCalendar;

@doc("Корректный email гостя (валидируется при создании бронирования)")
@pattern("^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Za-z]{2,}$")
scalar EmailString extends string;

@doc("Длительность события в минутах: целое число от 15 до 480")
@minValue(15)
@maxValue(480)
scalar DurationMinutes extends int32;

@doc("Дата в формате YYYY-MM-DD (в таймзоне владельца)")
@pattern("^\\d{4}-\\d{2}-\\d{2}$")
scalar DateString extends string;

@doc("График одного дня недели")
model DaySchedule {
  @doc("Является ли день рабочим (false — выходной)")
  isWorking: boolean;

  @doc("Начало рабочего дня в формате HH:MM")
  start: string;

  @doc("Конец рабочего дня в формате HH:MM")
  end: string;
}

@doc("График работы владельца по дням недели")
model Schedule {
  monday: DaySchedule;
  tuesday: DaySchedule;
  wednesday: DaySchedule;
  thursday: DaySchedule;
  friday: DaySchedule;
  saturday: DaySchedule;
  sunday: DaySchedule;
}

@doc("Профиль владельца календаря")
model Owner {
  @doc("Идентификатор владельца")
  id: string;

  @doc("Имя владельца")
  name: string;

  @doc("График работы владельца")
  schedule: Schedule;
}

@doc("Событие владельца — тип записи, который может забронировать гость")
model Event {
  @doc("Идентификатор события")
  id: string;

  @doc("Идентификатор владельца")
  ownerId: string;

  @doc("Название события")
  title: string;

  @doc("Описание события")
  description: string;

  @doc("Длительность события в минутах")
  durationMinutes: DurationMinutes;
}

@doc("Данные для создания события")
model EventCreate {
  @doc("Название события")
  title: string;

  @doc("Описание события")
  description: string;

  @doc("Длительность события в минутах (15–480)")
  durationMinutes: DurationMinutes;
}

@doc("Свободный слот времени")
model Slot {
  @doc("Начало слота (UTC, ISO-8601)")
  startAt: utcDateTime;

  @doc("Конец слота (UTC, ISO-8601)")
  endAt: utcDateTime;
}

@doc("Бронирование гостя")
model Booking {
  @doc("Идентификатор бронирования")
  id: string;

  @doc("Идентификатор события")
  eventId: string;

  @doc("Дата бронирования в таймзоне владельца (YYYY-MM-DD)")
  date: DateString;

  @doc("Начало бронирования (UTC, ISO-8601)")
  startAt: utcDateTime;

  @doc("Конец бронирования (UTC, ISO-8601)")
  endAt: utcDateTime;

  @doc("Имя гостя")
  guestName: string;

  @doc("Email гостя")
  guestEmail: EmailString;

  @doc("Дата и время создания бронирования (UTC)")
  createdAt: utcDateTime;
}

@doc("Данные для создания бронирования на выбранный слот")
model BookingCreate {
  @doc("Имя гостя")
  name: string;

  @doc("Email гостя")
  email: EmailString;

  @doc("Начало выбранного слота (UTC, ISO-8601)")
  startAt: utcDateTime;
}
