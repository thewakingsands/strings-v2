## xivstrings Go API

This Go project exposes a small HTTP API over the JSON data exported by
[ixion](https://github.com/thewakingsands/ixion)'s strings export. Data is
loaded from the
[ixion releases](https://github.com/thewakingsands/ixion/releases): on startup
the server fetches the latest release and, if needed, downloads `strings.zip`,
extracts it, and builds a search index. Queries use the current version's data.

### Building and running

From the `xivstrings` directory:

```bash
go run .
```

By default the server:

- Listens on `127.0.0.1:8080`
- Uses `data/` as the root directory for app data

Override with flags:

```bash
go run . -addr=":8090" -data="/path/to/data"
```

### Data directory layout

The `-data` path is the root for all app data. The server expects or creates:

- `data/version` - text file with the current version, for example `publish-20260303-8b409c8`
- `data/strings/<version>/` - extracted JSON files from `strings.zip`
- `data/index/<version>/` - Bleve search index for that version

On first run, or when the version changes, the server fetches the latest
[ixion release](https://github.com/thewakingsands/ixion/releases/latest),
downloads `strings.zip`, extracts to `data/strings/<version>/`, and builds the
index under `data/index/<version>/`.

### Version and update

- `GET /api/version` returns the current data version and the latest update status
- `POST /api/version?token=...` starts an async update job and returns `202 Accepted`
- Full update API details: [docs/update-api.md](docs/update-api.md)

Example:

```bash
# Optional: set token so POST /api/version can trigger updates
export XIVSTRINGS_UPDATE_TOKEN=your-secret-token
go run .

# Query current version and update status
curl http://127.0.0.1:8080/api/version

# Trigger async update
curl -X POST "http://127.0.0.1:8080/api/version?token=your-secret-token"
```

### Data format

Each JSON file under the strings directory contains an array of items:

```json
{
  "sheet": "AchievementKind",
  "rowId": "1",
  "values": {
    "en": "Battle",
    "ja": "バトル",
    "chs": "战斗"
  }
}
```

### HTTP APIs

- Search strings
  - Endpoint: `GET /api/search`
  - Query parameters: `lang` required, `q` required, `sheet` optional, `offset` optional, `limit` optional
  - Response: JSON with matching items and meta fields such as `total` and `elapsed`

- Get items by sheet
  - Endpoint: `GET /api/items`
  - Query parameters: `sheet` required, `offset` optional, `limit` optional
  - Response: JSON with items for the sheet and meta

- Version
  - `GET /api/version`: current data version and update status
  - `POST /api/version?token=...`: start async update from the latest ixion release
  - See [docs/update-api.md](docs/update-api.md) for the full contract
