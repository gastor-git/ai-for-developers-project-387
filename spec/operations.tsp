import "@typespec/http";
import "@typespec/rest";

using TypeSpec.Http;
using TypeSpec.Rest;

namespace BookingCalendar;

@doc("Операции владельца календаря")
@route("owners/{ownerId}")
@tag("Owner")
namespace OwnerResource {
  @doc("Получить профиль владельца и его график работы")
  @summary("Профиль владельца и график работы")
  @get
  op getOwner(
    @path ownerId: string,
  ): Owner | BadRequestError | NotFoundError;

  @doc("Обновить график работы владельца")
  @summary("Обновление графика работы владельца")
  @patch
  @route("schedule")
  op updateSchedule(
    @path ownerId: string,
    @body schedule: Schedule,
  ): Owner | BadRequestError | NotFoundError;

  @doc("Операции над событиями владельца")
  @route("events")
  @tag("Events")
  namespace EventResource {
    @doc("Получить список событий владельца")
    @summary("Список событий владельца")
    @get
    op listEvents(
      @path ownerId: string,
    ): Event[] | BadRequestError | NotFoundError;

    @doc("Создать новое событие владельца")
    @summary("Создание события")
    @post
    op createEvent(
      @path ownerId: string,
      @body event: EventCreate,
    ): Event | BadRequestError | NotFoundError;

    @doc("Операции над конкретным событием")
    @route("{eventId}")
    namespace SingleEventResource {
      @doc("Получить страницу события для гостя")
      @summary("Страница события для гостя")
      @get
      op getEvent(
        @path ownerId: string,
        @path eventId: string,
      ): Event | BadRequestError | NotFoundError;

      @doc("Свободные слоты события на указанную дату")
      @summary("Свободные слоты на дату")
      @route("slots")
      @tag("Slots")
      namespace SlotResource {
        @doc("Получить свободные слоты события на дату в окне записи")
        @summary("Получить свободные слоты события на дату")
        @get
        op getSlots(
          @path ownerId: string,
          @path eventId: string,
          @query date: DateString,
        ): Slot[] | BadRequestError | NotFoundError;
      }

      @doc("Бронирования события")
      @route("bookings")
      @tag("Bookings")
      namespace EventBookingResource {
        @doc("Создать бронирование на свободный слот события")
        @summary("Создание бронирования")
        @post
        op createBooking(
          @path ownerId: string,
          @path eventId: string,
          @body booking: BookingCreate,
        ): Booking | BadRequestError | NotFoundError | ConflictError;
      }
    }
  }

  @doc("Все бронирования владельца")
  @route("bookings")
  @tag("Bookings")
  namespace OwnerBookingResource {
    @doc("Получить список всех бронирований владельца")
    @summary("Все бронирования владельца")
    @get
    op listBookings(
      @path ownerId: string,
    ): Booking[] | BadRequestError | NotFoundError;
  }
}
