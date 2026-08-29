# schema-api

A schema-driven REST API server written in Go. Define your database tables in a
simple JSON schema file, and `schema-api` creates an in-memory SQLite database,
seeds it with realistic sample data, and exposes full CRUD endpoints backed by
schema-aware validation. You can also declare explicit mock endpoints that serve
templated JSON responses either on their own or alongside tables.

## Features

- **Schema-driven tables**: describe tables and columns in a single JSON file
- **In-memory SQLite**: tables are created automatically on startup
- **Realistic sample data**: built-in generators for names, emails, phones,
  addresses, credit cards, IBANs, IP addresses, coordinates, lorem text,
  UUIDs, and more
- **Foreign keys**: seeds rows in dependency order and honors FK references
- **Full CRUD API**: list, get, create, update, and delete rows
- **Input validation**: type checks, numeric ranges, string lengths, regex
  patterns, required fields, and defaults
- **Pagination & sorting**: `page`, `limit`, `sort`, and `order` query params
- **Constraint handling**: returns proper HTTP codes for unique and foreign
  key violations
- **Mock endpoints**: declare explicit routes (method, path, status, headers,
  and templated responses) alongside or instead of tables
- **Table-backed endpoints**: declare `GET`-only endpoints that run real SQL
  queries (joins, filters, sorting, limits) against your tables and return
  custom-shaped JSON built from `{{table.column}}` references
- **Response templating**: generate fields with the shared fake generators or
  interpolate path, query, header, and request-body values into responses
- **Cross-platform builds**: versioned archives for Linux, macOS, and Windows
  (amd64 and arm64) via the `Makefile`

## Requirements

- [Go](https://go.dev/dl/) 1.26.1 or newer

## Install & Build

Build a single binary:

```bash
go build -o schema-api .
```

Build versioned archives for all supported platforms (Linux, macOS, Windows ×
amd64/arm64) into `dist/`:

```bash
make build
```

Other useful targets:

```bash
make test        # go test ./...
make test-race   # go test -race ./...
make coverage    # run tests with a coverage profile and HTML report in coverage/
make clean       # remove dist/
```

## Usage

```bash
./schema-api [flags]
```

| Flag           | Default      | Description                                                                  |
| -------------- | ------------ | ---------------------------------------------------------------------------- |
| `-schema`      | _(required)_ | Path to the JSON schema file                                                 |
| `-rows`        | `10`         | Number of fake rows to seed per table (use `0` to skip seeding; tables only) |
| `-port`        | `8080`       | Server port                                                                  |
| `-cors-origin` | `*`          | Value for the `Access-Control-Allow-Origin` response header                  |
| `-version`     | `false`      | Print version info and exit                                                  |

The `-schema` flag is required; the server exits with an error if it is omitted.

Examples:

```bash
./schema-api -schema schema.json            # seed 10 rows, port 8080
./schema-api -schema my-schema.json         # custom schema
./schema-api -schema schema.json -rows 25 -port 9090
./schema-api -schema schema.json -rows 0    # start without seeding
./schema-api -version
# schema-api v1.0.0 (commit <hash>, built <date>)
```

On startup, each table is created and seeded (and mock endpoints are
registered), then the server prints the endpoints it exposes and runs until
interrupted (Ctrl-C sends a graceful shutdown):

```
table users created.
table posts created.
table comments created.
seeding 10 rows per table...
seeding complete.
mock endpoints registered.
Endpoints
METHOD  PATH                 SOURCE  STATUS
------  -------------------  ------  ------
GET     /users               crud    200
GET     /users/{id}          crud    200
POST    /users               crud    201
PUT     /users/{id}          crud    200
DELETE  /users/{id}          crud    204
...
GET     /users/{id}/profile  mock    200
POST    /echo                mock    201
GET     /authors/{id}/posts  table_endpoint  200
Server running on http://localhost:8080
```

## Schema definition

A schema file is a JSON object with a `tables` array, an optional `endpoints`
array, and an optional `table_endpoints` array. The sections are independent and
can coexist: the server enables whatever sections the schema declares, and exits
with an error if `tables` and `endpoints` are both empty. Each table has a
`name` and an array of `columns`. Every table automatically gets an `id INTEGER
PRIMARY KEY AUTOINCREMENT` column.

```json
{
  "tables": [
    {
      "name": "users",
      "columns": [
        {
          "name": "name",
          "type": "string",
          "required": true,
          "min_length": 2,
          "max_length": 50
        },
        { "name": "email", "type": "string", "unique": true },
        { "name": "age", "type": "int", "min": 18, "max": 120, "default": 18 },
        { "name": "score", "type": "float", "min": 0.0, "max": 100.0 },
        { "name": "active", "type": "bool", "default": true },
        { "name": "joined", "type": "datetime", "default": "now" },
        { "name": "address", "type": "string", "format": "address" }
      ]
    },
    {
      "name": "posts",
      "columns": [
        {
          "name": "title",
          "type": "string",
          "required": true,
          "format": "title"
        },
        { "name": "body", "type": "string", "format": "paragraph" },
        {
          "name": "user_id",
          "type": "int",
          "foreign_key": { "table": "users", "column": "id" }
        }
      ]
    }
  ]
}
```

### Column properties

| Property      | Type   | Description                                                                                                  |
| ------------- | ------ | ------------------------------------------------------------------------------------------------------------ |
| `name`        | string | Column name (required)                                                                                       |
| `type`        | string | One of `string`, `int`, `float`, `bool`, `datetime` (required)                                               |
| `required`    | bool   | Column must be provided on create                                                                            |
| `unique`      | bool   | Values must be unique (DB `UNIQUE` constraint)                                                               |
| `min`         | number | Minimum value for `int`/`float` columns                                                                      |
| `max`         | number | Maximum value for `int`/`float` columns                                                                      |
| `min_length`  | int    | Minimum length for `string` columns                                                                          |
| `max_length`  | int    | Maximum length for `string` columns                                                                          |
| `regex`       | string | Pattern that `string` values must match                                                                      |
| `format`      | string | Hints the data generator (see formats below)                                                                 |
| `default`     | any    | Default value applied when the column is omitted (use `"now"` for `datetime` to default to the current time) |
| `foreign_key` | object | `{ "table": "...", "column": "..." }` reference                                                              |

### Supported types

| Type       | SQLite type | Validation                                      |
| ---------- | ----------- | ----------------------------------------------- |
| `string`   | `TEXT`      | Length and regex checks                         |
| `int`      | `INTEGER`   | Numeric range checks                            |
| `float`    | `REAL`      | Numeric range checks                            |
| `bool`     | `INTEGER`   | Must be a boolean                               |
| `datetime` | `TEXT`      | RFC3339, `YYYY-MM-DD HH:MM:SS`, or `YYYY-MM-DD` |

### Data generation formats

When seeding, the generator is chosen from the `format` property, or inferred
from the column name (e.g. a column named `email` generates emails).

| Format            | Description                          | Also inferred from                                 |
| ----------------- | ------------------------------------ | -------------------------------------------------- |
| `name`            | Full name                            | `full_name`                                        |
| `firstname`       | First name                           | `first_name`                                       |
| `lastname`        | Last name                            | `last_name`                                        |
| `username`        | Username                             | `user_name`                                        |
| `email`           | Email address                        | `user_email`                                       |
| `phone`           | Phone number                         | -                                                  |
| `address`         | Street address                       | -                                                  |
| `city`            | City                                 | -                                                  |
| `country`         | Country                              | -                                                  |
| `url`             | URL                                  | -                                                  |
| `uuid`            | UUID v4                              | -                                                  |
| `credit_card`     | 16-digit card number (Luhn-valid)    | `card_number`                                      |
| `hex_color`       | Hex color code (e.g. `#A1B2C3`)      | `color`                                            |
| `ipv4`            | IPv4 address                         | `ip_address`                                       |
| `ipv6`            | IPv6 address                         | -                                                  |
| `mac_address`     | MAC address                          | -                                                  |
| `mime_type`       | MIME type (e.g. `application/json`)  | -                                                  |
| `file_extension`  | File extension including dot         | `file_ext`, `extension`                            |
| `currency_amount` | Dollar amount (e.g. `$123.45`)       | `price`, `amount`, `total`                         |
| `product_name`    | Fake product name                    | -                                                  |
| `slug`            | Hyphenated URL slug                  | `url_slug`                                         |
| `word`            | Single lorem word                    | -                                                  |
| `isbn`            | ISBN-13 with valid check digit       | -                                                  |
| `lat`             | Latitude between `-90` and `90`      | `latitude`                                         |
| `lng`             | Longitude between `-180` and `180`   | `longitude`                                        |
| `timezone`        | IANA timezone (e.g. `Europe/Madrid`) | `tz`                                               |
| `job_title`       | Job title                            | -                                                  |
| `company`         | Company name                         | `company_name`                                     |
| `iban`            | IBAN with mod-97 check digits        | `account_number`                                   |
| `date`            | Date in `YYYY-MM-DD` format          | `birth_date`, `dob`                                |
| `lorem`           | Lorem ipsum phrase                   | `description`, `bio`, `summary`, `body`, `content` |
| `sentence`        | Capitalized sentence ending in `.`   | -                                                  |

Matching is substring-based, so a column named `card_number` resolves to the
`credit_card` generator. Use type `datetime` for RFC3339 timestamps; `date`
produces a plain `YYYY-MM-DD` string.

## Mock endpoints

In addition to (or instead of) tables, you can declare an `endpoints` array.
Each entry defines an explicit route that serves a generated JSON response,
no database involved.

```json
{
  "endpoints": [
    {
      "method": "GET",
      "path": "/users/{id}/stats",
      "status": 200,
      "headers": { "X-Mock": "true" },
      "response": {
        "user_id": "{{path.id}}",
        "page": "{{query.page}}",
        "name": { "type": "string", "min_length": 5, "max_length": 20 },
        "age": { "type": "int", "min": 18, "max": 80 },
        "active": { "type": "bool" },
        "joined": { "type": "datetime" },
        "score": { "type": "float", "min": 0, "max": 100 },
        "tags": {
          "type": "array",
          "count": 5,
          "items": { "type": "string", "max_length": 10 }
        },
        "profile": { "bio": { "type": "string", "max_length": 120 } },
        "fixed": "literal value"
      }
    }
  ]
}
```

### Endpoint properties

| Property   | Type              | Description                                                   |
| ---------- | ----------------- | ------------------------------------------------------------- |
| `method`   | string (required) | `GET`, `POST`, `PUT`, `PATCH`, or `DELETE` (case-insensitive) |
| `path`     | string (required) | Starts with `/`; supports Go 1.22 wildcards like `{id}`       |
| `status`   | int               | Response status code, defaults to `200`                       |
| `headers`  | object            | Static response headers                                       |
| `response` | object (required) | Response template (see below)                                 |

Example request/response for the schema above:

```bash
curl -s "http://localhost:8080/users/42/stats?page=3" -H "X-Mock: true"
```

```json
{
  "user_id": "42",
  "page": "3",
  "name": "Andres Arias",
  "age": 54,
  "active": true,
  "joined": "2025-04-02T14:08:11Z",
  "score": 63.29,
  "tags": ["zqm3Xw", "rTb1", "cYq8aH2", "Wx", "oPj0R"],
  "profile": { "bio": "R9f3eVhQ" },
  "fixed": "literal value"
}
```

### Response templates

The `response` value is walked recursively:

- **Literals**: plain strings, numbers, booleans, `null`, and arrays are
  returned as-is.
- **Nested objects**: objects without a `type` key are template objects,
  walked recursively (nested literals and `{{...}}` both work).
- **Generator specs**: objects with a `type` key generate a value via the
  shared fake generator. Length/range keys use snake_case (`min_length`,
  `max_length`), matching the column format. Supported spec types:

  | Type       | Extra keys                                   |
  | ---------- | -------------------------------------------- |
  | `string`   | `min_length`, `max_length`, `format`         |
  | `int`      | `min`, `max`                                 |
  | `float`    | `min`, `max`                                 |
  | `bool`     | -                                            |
  | `datetime` | -                                            |
  | `array`    | `count`, `items` (nested spec or template)   |
  | `object`   | `properties` (map of nested specs/templates) |

  A top-level generator spec (e.g. `{ "type": "array", "count": 3, "items":
{...} }`) returns a JSON array.

  String specs without an explicit `format` inherit the name heuristics used
  for tables, e.g. `"email": { "type": "string" }` generates an email, and
  `"user_name"`/`"first_name"`/`"last_name"` generate usernames and names.
  An explicit `format` always wins.

- **Interpolation**: strings containing `{{...}}` are interpolated:

  | Source            | Description                                                       |
  | ----------------- | ----------------------------------------------------------------- |
  | `{{path.name}}`   | URL wildcard value (e.g. `{{path.id}}` for `/users/{id}`)         |
  | `{{query.name}}`  | Query parameter value                                             |
  | `{{header.name}}` | Request header, use the lowercase name (e.g. `{{header.x-mock}}`) |
  | `{{body.name}}`   | Value from the JSON request body (POST/PUT/PATCH echo)            |
  | `{{now}}`         | Current time (RFC3339)                                            |

  Missing keys resolve to an empty string; interpolated values are always
  strings.

Note: an object field literally named `type` cannot be expressed, it is
reserved for generator specs.

### Route precedence & conflicts

CRUD registers `GET /{table}`, `GET /{table}/{id}`, `POST /{table}`,
`PUT /{table}/{id}`, and `DELETE /{table}/{id}`. Go's `ServeMux` prefers the
more specific pattern, so a mock endpoint can shadow a CRUD route (e.g.
`GET /users` beats `GET /{table}`; `GET /users/{id}/stats` beats
`GET /{table}/{id}`). Two mock endpoints with the same `method+path`, or a
genuinely conflicting pattern where neither is more specific (e.g.
`GET /{thing}/{id}` vs CRUD's `GET /{table}/{id}`), are startup errors.

To avoid conflicts:

- Use a literal first segment for mock paths (`/users/{id}/stats`), never a
  wildcard mirroring CRUD's `{table}`.
- Make at least one segment literal or differ in segment count.
- Keep table names free for CRUD; anchor mock routes under a literal table
  name or dedicated prefix.

## Table endpoints

In addition to CRUD and mock endpoints, you can declare a `table_endpoints`
array. Each entry defines a `GET`-only route that runs a **real SQL query**
against your tables and shapes the result rows into a custom JSON response.
Unlike mock endpoints, the data comes from the database; unlike CRUD, the
response shape is entirely up to you.

```json
{
  "table_endpoints": [
    {
      "method": "GET",
      "path": "/authors/{id}/posts",
      "status": 200,
      "headers": { "X-Source": "database" },
      "tables": ["authors", "posts"],
      "joins": [
        {
          "type": "INNER",
          "on": { "local": "authors.id", "foreign": "posts.author_id" }
        }
      ],
      "where": ["authors.id = {{path.id}}", "posts.status = 'published'"],
      "order_by": "posts.published_at DESC",
      "limit": 20,
      "response": {
        "author": "{{authors.name}}",
        "author_email": "{{authors.email}}",
        "posts": {
          "type": "array",
          "items": {
            "title": "{{posts.title}}",
            "slug": "{{posts.slug}}",
            "view_count": "{{posts.view_count}}"
          }
        }
      }
    }
  ]
}
```

### Table endpoint properties

| Property   | Type              | Description                                                                                          |
| ---------- | ----------------- | ---------------------------------------------------------------------------------------------------- |
| `method`   | string (required) | Must be `GET` (case-insensitive)                                                                     |
| `path`     | string (required) | Starts with `/`; supports Go 1.22 wildcards like `{id}`                                              |
| `status`   | int               | Response status code, defaults to `200`                                                              |
| `headers`  | object            | Static response headers                                                                              |
| `tables`   | array (required)  | Tables to query (≥1). Must reference tables defined in `tables`                                      |
| `joins`    | array             | Explicit joins. If omitted, inferred from foreign keys between consecutive `tables`                   |
| `where`    | array             | SQL `WHERE` conditions; supports `{{path.name}}`, `{{query.name}}`, `{{header.name}}` interpolation |
| `order_by` | string            | SQL `ORDER BY` clause (e.g. `"posts.id DESC"`)                                                       |
| `limit`    | int               | SQL `LIMIT` value                                                                                    |
| `response` | object (required) | Response template (see below)                                                                        |

### Response shaping

The `response` object maps output keys to values:

- **Database values**: a string of the form `{{table.column}}` is replaced with
  that column from the query result. A scalar reference outside an array uses
  the first result row's value.
- **Arrays of rows**: an object with `"type": "array"` and an `items` template
  collects one object per result row into a JSON array. Inside `items`,
  `{{table.column}}` references resolve to each row's value.
- **Generator specs**: objects with a `type` key generate fake data via the
  shared generators (e.g. `{ "type": "string", "max_length": 12 }`), the same
  as mock endpoint response templates. These are not read from the database.
- **Literals / nested objects**: plain strings, numbers, booleans, and nested
  objects behave as in mock endpoint templates.

Example request/response for the schema above:

```bash
curl -s "http://localhost:8080/authors/42/posts"
```

```json
{
  "author": "Andres Arias",
  "author_email": "andres@example.com",
  "posts": [
    { "title": "Hello World", "slug": "hello-world", "view_count": 1337 },
    { "title": "Second Post", "slug": "second-post", "view_count": 42 }
  ]
}
```

### Joins

If `joins` is omitted, a join is inferred between each consecutive pair of
`tables` using foreign keys (an `INNER JOIN` on `<parent>.id = <child>.<fk>`).
If no foreign key relationship exists, the schema fails validation. Use the
explicit `joins` array to override or extend joins:

| Property | Type   | Description                                                       |
| -------- | ------ | ----------------------------------------------------------------- |
| `type`   | string | `INNER`, `LEFT`, `RIGHT`, or `CROSS` (defaults to `INNER`)        |
| `on`     | object | `{ "local": "table.column", "foreign": "table.column" }`          |

All referenced tables and columns must exist in `tables`.

### Route precedence & conflicts

Table endpoints follow the same `ServeMux` precedence rules as mock endpoints
(see above): a more specific pattern wins, so a table endpoint can shadow a
CRUD route. A table endpoint that collides with another route on the same
`method+path` (another table endpoint, a mock endpoint, or a CRUD route) is a
**startup error**. Keep the first path segment literal (e.g.
`/authors/{id}/posts`) rather than mirroring CRUD's `{table}` wildcard.

## API endpoints

The CRUD routes below apply to tables defined in the `tables` array. All routes
are mounted under `/{table}`, where `{table}` matches a table name in your
schema. Unrecognized tables return `404`.

### List rows

```
GET /{table}?page=1&limit=20&sort=id&order=asc
```

- `page`: page number, defaults to `1`
- `limit`: rows per page, defaults to `20` (max `100`)
- `sort`: column to sort by, defaults to `id`
- `order`: `asc` or `desc`, defaults to `asc`

Response headers:

- `X-Total-Count`: total number of rows
- `X-Page`: current page
- `X-Limit`: page size

```bash
curl "http://localhost:8080/users?page=2&limit=10&sort=age&order=desc"
```

### Get a row

```
GET /{table}/{id}
```

Returns the row, or `404` if it doesn't exist.

```bash
curl http://localhost:8080/users/1
```

### Create a row

```
POST /{table}
```

Body is a JSON object with the columns to set. Unrecognized fields are ignored;
fields are validated against the schema.

- `201 Created`: returns the created row
- `422 Unprocessable Entity`: validation errors (`{ "errors": [...] }`)
- `409 Conflict`: unique constraint violation
- `400 Bad Request`: foreign key violation or invalid JSON

```bash
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name": "Jane Doe", "email": "jane@example.com", "age": 30}'
```

### Update a row

```
PUT /{table}/{id}
```

Body is a JSON object with the columns to change (partial updates allowed).
Same validation and error codes as create, plus `404` if the row doesn't exist.

```bash
curl -X PUT http://localhost:8080/users/1 \
  -H "Content-Type: application/json" \
  -d '{"age": 31}'
```

### Delete a row

```
DELETE /{table}/{id}
```

- `204 No Content`: deleted
- `404 Not Found`: row doesn't exist
- `409 Conflict`: row is referenced by other rows (foreign key)

```bash
curl -X DELETE http://localhost:8080/users/1
```
