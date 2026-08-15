# schema-api

A schema-driven REST API server written in Go. Define your database tables in a
simple JSON schema file, and `schema-api` creates an in-memory SQLite database,
seeds it with realistic sample data, and exposes full CRUD endpoints backed by
schema-aware validation.

## Features

- **Schema-driven tables** — describe tables and columns in a single JSON file
- **In-memory SQLite** — tables are created automatically on startup
- **Realistic sample data** — built-in generators for names, emails, phones,
  addresses, cities, countries, URLs, UUIDs, and more
- **Foreign keys** — seeds rows in dependency order and honors FK references
- **Full CRUD API** — list, get, create, update, and delete rows
- **Input validation** — type checks, numeric ranges, string lengths, regex
  patterns, required fields, and defaults
- **Pagination & sorting** — `page`, `limit`, `sort`, and `order` query params
- **Constraint handling** — returns proper HTTP codes for unique and foreign
  key violations
- **Cross-platform builds** — versioned archives for Linux, macOS, and Windows
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
make test    # go test ./...  # test will be added later :)
make clean   # remove dist/
```

## Usage

```bash
./schema-api [flags]
```

| Flag       | Default      | Description                                                     |
| ---------- | ------------ | --------------------------------------------------------------- |
| `-schema`  | _(required)_ | Path to the JSON schema file                                    |
| `-rows`    | `10`         | Number of fake rows to seed per table (use `0` to skip seeding) |
| `-port`    | `8080`       | Server port                                                     |
| `-version` | `false`      | Print version info and exit                                     |

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

On startup, each table is created and seeded, then the server runs until
interrupted (Ctrl-C sends a graceful shutdown):

```
table users created.
table posts created.
table comments created.
seeding 10 rows per table...
seeding complete.
Server running on http://localhost:8080
```

## Schema definition

A schema file is a JSON object with a `tables` array. Each table has a `name`
and an array of `columns`. Every table automatically gets an
`id INTEGER PRIMARY KEY AUTOINCREMENT` column.

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

| Format      | Description    |
| ----------- | -------------- |
| `name`      | Full name      |
| `firstname` | First name     |
| `lastname`  | Last name      |
| `username`  | Username       |
| `email`     | Email address  |
| `phone`     | Phone number   |
| `address`   | Street address |
| `city`      | City           |
| `country`   | Country        |
| `url`       | URL            |
| `uuid`      | UUID v4        |

## API endpoints

All routes are mounted under `/{table}`, where `{table}` matches a table name in
your schema. Unrecognized tables return `404`.

### List rows

```
GET /{table}?page=1&limit=20&sort=id&order=asc
```

- `page` — page number, defaults to `1`
- `limit` — rows per page, defaults to `20` (max `100`)
- `sort` — column to sort by, defaults to `id`
- `order` — `asc` or `desc`, defaults to `asc`

Response headers:

- `X-Total-Count` — total number of rows
- `X-Page` — current page
- `X-Limit` — page size

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
fields validated against the schema.

- `201 Created` — returns the created row
- `422 Unprocessable Entity` — validation errors (`{ "errors": [...] }`)
- `409 Conflict` — unique constraint violation
- `400 Bad Request` — foreign key violation or invalid JSON

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

- `204 No Content` — deleted
- `404 Not Found` — row doesn't exist
- `409 Conflict` — row is referenced by other rows (foreign key)

```bash
curl -X DELETE http://localhost:8080/users/1
```
