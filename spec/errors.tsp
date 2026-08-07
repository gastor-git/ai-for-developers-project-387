import "@typespec/http";

using TypeSpec.Http;

namespace BookingCalendar;

@doc("Тело ошибки")
model ApiErrorBody {
  @doc("Машиночитаемый код ошибки")
  code: string;

  @doc("Человекочитаемое описание ошибки")
  message: string;
}

@doc("Невалидный запрос: нарушена валидация полей или параметров")
@error
model BadRequestError {
  @statusCode statusCode: 400;

  @doc("Описание ошибки")
  error: ApiErrorBody;
}

@doc("Запрошенный ресурс не найден")
@error
model NotFoundError {
  @statusCode statusCode: 404;

  @doc("Описание ошибки")
  error: ApiErrorBody;
}

@doc("Конфликт: слот уже занят или нарушены ограничения записи")
@error
model ConflictError {
  @statusCode statusCode: 409;

  @doc("Описание ошибки")
  error: ApiErrorBody;
}
