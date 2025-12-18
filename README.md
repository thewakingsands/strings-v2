## xivstrings Go API

This Go project exposes a small HTTP API over the JSON data exported by ixion's `StringsExporter`.

The server loads all `*.json` files from the `data/` directory into memory at startup.

### Building and running

From the `xivstrings` directory:

```bash
go run .
```

By default the server:

- Listens on `:8080`
- Reads JSON files from the `data/` directory next to `main.go`

You can override these with flags:

```bash
go run . -addr=":8090" -data="path/to/data"
```

### Data format

Each JSON file in `data/` should contain an array of items:

```json
{
  "sheet": "AchievementKind",
  "rowId": "1",
  "field": "Name",
  "values": {
    "en": "Battle",
    "ja": "バトル",
    "chs": "战斗"
  }
}
```

### HTTP APIs

- **Search strings**

  - **Endpoint**: `GET /search`
  - **Query parameters**:
    - `lang` (required): language code, e.g. `en`, `ja`, `chs`
    - `q` (required): substring to search for in the given language
    - `sheet` (optional): restrict search to a specific sheet name
    - `limit` (optional): maximum number of results (default 100, max 1000)
  - **Response**: JSON array of matching items, each containing `sheet`, `rowId`, `field`, and `values` (all languages).

- **Get items for a row**

  - **Endpoint**: `GET /items`
  - **Query parameters**:
    - `sheet` (required): sheet name
    - `rowId` (required): row id (e.g. `"1"` or `"1.0"`)
  - **Response**: JSON array of all items for the given sheet and rowId.


