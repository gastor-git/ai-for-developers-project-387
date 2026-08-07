import "@typespec/http";
import "@typespec/openapi";
import "@typespec/rest";
import "./models.tsp";
import "./errors.tsp";
import "./operations.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@service(#{ title: "Booking Calendar API" })
@info(#{ version: "1.0.0" })
@doc(
  "API упрощённого аналога cal.com: владелец календаря настраивает профиль и создаёт события, гости бронируют слоты без регистрации и авторизации.",
)
@useAuth(NoAuth)
@server("/api", "Базовый путь API")
@tagMetadata(#[
  #{ name: "Owner", description: "Профиль владельца календаря" },
  #{ name: "Events", description: "События владельца и их страницы" },
  #{ name: "Slots", description: "Свободные слоты для бронирования" },
  #{ name: "Bookings", description: "Бронирования гостей" },
])
namespace BookingCalendar;
