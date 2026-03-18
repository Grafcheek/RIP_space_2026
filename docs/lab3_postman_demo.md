# Lab 3 Postman demo

## Что импортировать в Postman

- коллекцию: `docs/postman/rip_space_lab3.postman_collection.json`
- базовый адрес сервера: `http://localhost:8080`

## Переменные коллекции

- `baseUrl` = `http://localhost:8080`
- `draftRequestId` = пусто перед стартом
- `createdServiceId` = пусто перед стартом
- `firstRouteId` = `1`
- `secondRouteId` = `2`
- `newUsername` = например `space_guest_2026`
- `newPassword` = например `orbital_pass_2026`

## Подготовка перед показом

1. Убедиться, что БД поднята и применены `db/init/001_schema.sql`, `db/init/002_seed.sql`.
2. Для Lab 3 один раз применить `db/init/003_lab3_api_columns.sql`.
3. Запустить сервер `go run ./cmd/app`.
4. В запросе `POST /api/services` в Postman выбрать локальные файлы для полей `image` и `video`.

## Порядок показа по ТЗ

### 1. GET список заявок с фильтрацией

- Запрос: `GET /api/requests?status=formed&from=2026-01-01T00:00:00Z&to=2026-12-31T23:59:59Z`
- Показываем, что список не содержит `draft` и `deleted`.

### 2. GET иконка корзины

- Запрос: `GET /api/requests/cart-icon`
- Если есть черновик, сохранить его `id` в `draftRequestId`.

### 3. DELETE введённую заявку, если есть

- Запрос: `DELETE /api/requests/{{draftRequestId}}`
- Если черновика нет, этот шаг можно проговорить как условный.

### 4. GET список услуг с фильтром

- Запрос: `GET /api/services?query=земля`

### 5. POST новая услуга с картинкой и видео

- Запрос: `POST /api/services`
- Тип: `form-data`
- Поля: `title`, `from_body`, `to_body`, `description`, `from_orbit_radius_km`, `to_orbit_radius_km`, `image`, `video`
- Из ответа сохранить `id` в `createdServiceId`.

### 6. POST добавить первую услугу в заявку

- Запрос: `POST /api/requests/draft/items`
- Тело: `{ "route_id": {{createdServiceId}} }`
- Из ответа сохранить `id` заявки в `draftRequestId`.

### 7. POST добавить другую услугу в заявку

- Запрос: `POST /api/requests/draft/items`
- Тело: `{ "route_id": {{secondRouteId}} }`

### 8. GET иконка корзины ещё раз

- Запрос: `GET /api/requests/cart-icon`
- Должно быть количество `2`.

### 9. GET одна заявка

- Запрос: `GET /api/requests/{{draftRequestId}}`
- Показываем, что у заявки теперь 2 услуги.

### 10. PUT изменить поле м-м

- Запрос: `PUT /api/requests/{{draftRequestId}}/items/{{createdServiceId}}`
- Тело:

```json
{
  "quantity": 2,
  "segment_order": 1,
  "segment_dry_mass_kg": 2600,
  "segment_isp_sec": 410
}
```

### 11. PUT изменить заявку

- Запрос: `PUT /api/requests/{{draftRequestId}}`
- Тело:

```json
{
  "spacecraft_dry_mass_kg": 3200,
  "engine_isp_sec": 430
}
```

### 12. PUT завершить введённую заявку и показать ошибку

- Запрос: `PUT /api/requests/{{draftRequestId}}/moderate`
- Тело:

```json
{
  "action": "complete"
}
```

- Ожидаемо приходит ошибка, так как завершать можно только `formed`.

### 13. PUT сформировать заявку

- Запрос: `PUT /api/requests/{{draftRequestId}}/form`
- Во время формирования пересчитывается `total_fuel_mass_kg`, а также поля м-м `delta_v_kms` и `fuel_mass_kg`.

### 14. PUT завершить сформированную заявку

- Запрос: `PUT /api/requests/{{draftRequestId}}/moderate`
- Тело:

```json
{
  "action": "complete"
}
```

### 15. POST зарегистрировать нового пользователя

- Запрос: `POST /api/users/register`
- Тело:

```json
{
  "username": "{{newUsername}}",
  "password": "{{newPassword}}"
}
```

## Как добрать до коллекции из 16 запросов

В самой коллекции лежат все 16 REST-методов приложения:

1. `GET /api/services`
2. `GET /api/services/:id`
3. `POST /api/services`
4. `POST /api/requests/draft/items`
5. `PUT /api/requests/:id/items/:routeId`
6. `DELETE /api/requests/:id/items/:routeId`
7. `GET /api/requests/cart-icon`
8. `GET /api/requests`
9. `GET /api/requests/:id`
10. `PUT /api/requests/:id`
11. `PUT /api/requests/:id/form`
12. `PUT /api/requests/:id/moderate`
13. `DELETE /api/requests/:id`
14. `POST /api/users/register`
15. `POST /api/users/login`
16. `POST /api/users/logout`

Во время демонстрации некоторые из них вызываются повторно: `GET /api/requests/cart-icon` и `POST /api/requests/draft/items`.

## Что показать через SQL select

### После удаления черновика

```sql
select id, status, created_by, formed_at, completed_at
from flight_requests
order by id desc;
```

### После создания новой услуги

```sql
select id, title, from_body, to_body, image_url, video_url
from transfer_routes
order by id desc
limit 5;
```

### После добавления услуг в заявку

```sql
select flight_request_id, route_id, quantity, segment_order, is_primary
from flight_request_routes
order by flight_request_id desc, segment_order asc;
```

### После изменения поля м-м

```sql
select flight_request_id, route_id, quantity, segment_order,
       segment_dry_mass_kg, segment_isp_sec, delta_v_kms, fuel_mass_kg
from flight_request_routes
where flight_request_id = {{draftRequestId}}
order by segment_order asc;
```

### После изменения заявки

```sql
select id, status, spacecraft_dry_mass_kg, engine_isp_sec, total_fuel_mass_kg
from flight_requests
where id = {{draftRequestId}};
```

### После формирования и завершения

```sql
select id, status, created_by, moderated_by, formed_at, completed_at, total_fuel_mass_kg
from flight_requests
where id = {{draftRequestId}};
```

### После регистрации пользователя

```sql
select id, username, is_moderator
from users
order by id desc
limit 5;
```

## Что сказать про модели

- `TransferRoute` — доменная модель услуги, таблица `transfer_routes`.
- `FlightRequest` — доменная модель заявки, таблица `flight_requests`.
- `FlightRequestRouteRow` — связь многие-ко-многим между заявкой и услугой, таблица `flight_request_routes`.
- `User` — доменная модель пользователя, таблица `users`.

## Что сказать про сериализаторы

В Go роль сериализаторов выполняют DTO-структуры и JSON/form binding:

- `MissionProfile` и `MissionRoute` — DTO ответа для одной заявки с услугами.
- `RequestListItem` — DTO ответа для списка заявок.
- `createServiceRequest`, `addToDraftRequest`, `updateMMRequest`, `updateRequestPayload`, `moderateRequestPayload`, `registerUserPayload` — структуры входных данных HTTP API.

Gin сериализует ответы через `ctx.JSON(...)`, а входные данные привязывает через `ctx.Bind(...)` и `ctx.BindJSON(...)`.
