## xivstrings Go API

This Go project exposes a small HTTP API over the JSON data exported by [ixion](https://github.com/thewakingsands/ixion)'s strings export. Data is loaded from the [ixion releases](https://github.com/thewakingsands/ixion/releases): on startup the server fetches the latest release and, if needed, downloads `strings.zip`, extracts it, and builds a search index. Queries use the current version's data.

### Building and running

From the `xivstrings` directory:

```bash
go run .
```

By default the server:

- Listens on `127.0.0.1:8080`
- Uses `data/` as the root directory for app data (see below)

Override with flags:

```bash
go run . -addr=":8090" -data="/path/to/data"
```

### Data directory layout

The `-data` path is the **root** for all app data. The server expects or creates:

- `data/version` — text file with the current version (e.g. `publish-20260303-8b409c8`)
- `data/strings/<version>/` — extracted JSON files from `strings.zip`
- `data/index/<version>/` — Bleve search index for that version

On first run (or when the version changes), the server will fetch the latest [ixion release](https://github.com/thewakingsands/ixion/releases/latest), download `strings.zip`, extract to `data/strings/<version>/`, and build the index under `data/index/<version>/`.

### Version and update

- **GET /api/version** — returns the current data version, e.g. `{"version":"publish-20260303-8b409c8"}`.
- **POST /api/version** — triggers a version check and, if the latest release differs from local, downloads and indexes the new data, then reloads the store. Requires a `token` query parameter that matches the environment variable `XIVSTRINGS_UPDATE_TOKEN`. If `XIVSTRINGS_UPDATE_TOKEN` is not set, POST returns `403` and update is not allowed.

Example:

```bash
# Optional: set token so POST /api/version can trigger updates
export XIVSTRINGS_UPDATE_TOKEN=your-secret-token
go run .

# Query current version (no auth)
curl http://127.0.0.1:8080/api/version

# Trigger update (with token)
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

- **Search strings**

  - **Endpoint**: `GET /api/search`
  - **Query parameters**:
    - `lang` (required): language code, e.g. `en`, `ja`, `chs`
    - `q` (required): substring to search for in the given language
    - `sheet` (optional): restrict search to a specific sheet name
    - `offset` (optional): offset of the first item to return (default 0)
    - `limit` (optional): maximum number of items to return (default 100, max 1000)
  - **Response**: JSON with matching items and meta (total, elapsed).

- **Get items by sheet**

  - **Endpoint**: `GET /api/items`
  - **Query parameters**:
    - `sheet` (required): sheet name
    - `offset` (optional): offset (default 0)
    - `limit` (optional): max items (default 100, max 1000)
  - **Response**: JSON with items for the sheet and meta.

- **Version**
  - **GET /api/version**: current data version.
  - **POST /api/version?token=...**: run update from ixion release (requires `XIVSTRINGS_UPDATE_TOKEN`).
