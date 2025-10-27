# Go Geolocation Service

Microservice that resolves IP addresses to geographic information using MaxMind's GeoLite2 City database. The project exposes a simple HTTP API, ships with a Cobra-powered CLI, and includes helpers to download and refresh the database on demand.

## Features

- REST endpoints for IP lookups, liveness, readiness, and database refresh.
- Service abstraction around MaxMind with hot reload after updates.
- Configurable refresh window, HTTP timeouts, and environment-driven settings.
- Multi-stage Docker build producing a tiny, static binary image.

## Quickstart

```bash
git clone https://github.com/thiagozs/geolocation-go.git
cd geolocation-go

# optional: copy sample env
cp .env.example .env  # if you maintain one

go run ./cmd/geolocation runserver --http=5000
```

The CLI reuses Cobra commands, so you can inspect the help at any time with `go run ./cmd/geolocation --help`.

### Configuration

| Variable | Description | Default |
| --- | --- | --- |
| `MODE` | `development` enables debug Gin mode; anything else switches to release mode | `development` |
| `MAXMIND_KEY` | GeoLite2 license key required to download database updates | _empty_ |
| `MAXMIND_DB_PATH` | Location of the GeoLite2-City `.mmdb` file | `db/GeoLite2-City.mmdb` |
| `MAXMIND_HTTP_TIMEOUT` | Timeout for MaxMind HTTP requests (`time.ParseDuration` format or seconds) | `30s` |
| `MAXMIND_REFRESH_INTERVAL` | Minimum interval before re-downloading the database (`time.ParseDuration` or seconds) | `24h` |

You can pass the same settings via CLI flags or environment variables that Viper understands (.env, shell, etc.).

## API Endpoints

| Method | Path | Description |
| --- | --- | --- |
| `GET /ip?address=1.1.1.1` | Returns GeoLite2 record for the provided IP address. |
| `GET /updatedb[?force=true]` | Downloads the latest GeoLite2 database when checksums differ. Requires `MAXMIND_KEY`. |
| `GET /healthz` | Simple liveness probe. |
| `GET /readiness` | Reports readiness based on database availability. |

Example response for `/ip`:

```json
{
  "data": {
    "Country": {
      "IsInEuropeanUnion": false,
      "ISOCode": "US"
    },
    "City": {
      "Names": {
        "en": "Nashville",
        "pt-BR": "Nashville"
      }
    },
    "Location": {
      "AccuracyRadius": 500,
      "Latitude": 36.0964,
      "Longitude": -86.8212,
      "TimeZone": "America/Chicago"
    },
    "IP": "4.4.4.4"
  }
}
```

## Updating the MaxMind Database

1. Obtain a GeoLite2 license key from [MaxMind](https://www.maxmind.com/en/accounts/current/license-key).
2. Export the key (`export MAXMIND_KEY=your_key`) or add it to your `.env`.
3. Trigger an update by calling `GET /updatedb`.  
   - The service compares checksums and skips when the local file is fresh.  
   - Use `GET /updatedb?force=true` to bypass the refresh interval and force a download.

After a successful update, the service reloads the reader transparently so subsequent requests use the new data.

## Running Tests

```bash
GOCACHE=$(mktemp -d) go test ./...
```

The suite covers downloader logic, service readiness flows, and HTTP handlers.

## Docker

Build and run with the bundled Dockerfile:

```bash
docker build -t geolocation:latest .
docker run --rm -p 5000:5000 \
  -e MAXMIND_KEY=your_license_key \
  geolocation:latest runserver --http=5000
```

The image sets `MAXMIND_DB_PATH=/app/db/GeoLite2-City.mmdb` by default. Mount a volume at `/app/db` if you want persistence across runs.

## Versioning and License

This project follows [Semantic Versioning](https://semver.org/). Check the repository tags for published versions.

Licensed under the terms described in [LICENSE](LICENSE).

© 2023 present Thiago Zilli Sarmento :heart:
