# Update API

The update API lets you trigger a background refresh of the local `xivstrings`
data from the latest `ixion` release, then poll for completion.

`POST /api/version` is asynchronous. It returns immediately with
`202 Accepted`, while the server downloads `strings.zip`, rebuilds the Bleve
index, and swaps the in-memory store in the background.

## Authentication

Updates are disabled unless the server is started with the
`XIVSTRINGS_UPDATE_TOKEN` environment variable.

Clients must pass the same value as the `token` query parameter:

```text
POST /api/version?token=your-secret-token
```

If the token is missing or invalid, the server rejects the request.

## Endpoints

### Get current version and update status

`GET /api/version`

Returns the currently active data version and the latest update job status.

Example response when idle:

```json
{
  "version": "publish-20260303-8b409c8",
  "update": {
    "state": "idle"
  }
}
```

Example response while running:

```json
{
  "version": "publish-20260303-8b409c8",
  "update": {
    "state": "running",
    "startedAt": "2026-05-11T13:06:40Z"
  }
}
```

Example response after success:

```json
{
  "version": "publish-20260510-deadbeef",
  "update": {
    "state": "success",
    "startedAt": "2026-05-11T13:06:40Z",
    "finishedAt": "2026-05-11T13:10:12Z",
    "version": "publish-20260510-deadbeef",
    "updated": true
  }
}
```

Example response after a no-op check:

```json
{
  "version": "publish-20260510-deadbeef",
  "update": {
    "state": "success",
    "startedAt": "2026-05-11T13:20:00Z",
    "finishedAt": "2026-05-11T13:20:01Z",
    "version": "publish-20260510-deadbeef",
    "updated": false
  }
}
```

Example response after failure:

```json
{
  "version": "publish-20260303-8b409c8",
  "update": {
    "state": "error",
    "startedAt": "2026-05-11T13:06:40Z",
    "finishedAt": "2026-05-11T13:07:05Z",
    "error": "fetch latest release: github api status 403: rate limit exceeded"
  }
}
```

### Trigger an update

`POST /api/version?token=...`

Starts an update job if one is not already running.

Example response when a new job starts:

```json
{
  "message": "update started",
  "update": {
    "state": "running",
    "startedAt": "2026-05-11T13:06:40Z"
  }
}
```

Example response when a job is already running:

```json
{
  "message": "update already in progress",
  "update": {
    "state": "running",
    "startedAt": "2026-05-11T13:06:40Z"
  }
}
```

## Update status fields

The `update` object can contain these fields:

- `state`: one of `idle`, `running`, `success`, or `error`
- `startedAt`: UTC timestamp in RFC 3339 format
- `finishedAt`: UTC timestamp in RFC 3339 format
- `version`: release version detected by the completed update job
- `updated`: `true` if a new version was downloaded and loaded, `false` if the latest version was already present
- `error`: failure message for `state = "error"`

## Typical client flow

1. Call `POST /api/version?token=...`.
2. If the server returns `202`, start polling `GET /api/version`.
3. Continue polling while `update.state` is `running`.
4. Stop when `update.state` becomes `success` or `error`.
5. Read `update.updated` to determine whether a new version was actually applied.

## Status codes

- `200 OK`: returned by `GET /api/version`
- `202 Accepted`: returned by `POST /api/version` when the request is accepted
- `400 Bad Request`: missing `token`
- `401 Unauthorized`: invalid `token`
- `403 Forbidden`: updates are disabled because `XIVSTRINGS_UPDATE_TOKEN` is not configured
- `405 Method Not Allowed`: unsupported HTTP method
- `500 Internal Server Error`: unexpected server-side error while reading the current version

## Curl examples

```bash
curl http://127.0.0.1:8080/api/version
```

```bash
curl -X POST "http://127.0.0.1:8080/api/version?token=your-secret-token"
```

```bash
watch -n 5 'curl -s http://127.0.0.1:8080/api/version'
```
