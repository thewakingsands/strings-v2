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
  - **Response**: JSON array of matching items, each containing `sheet`, `rowId`, and `values` (all languages).

- **Get items for a row**

  - **Endpoint**: `GET /api/items`
  - **Query parameters**:
    - `sheet` (required): sheet name
    - `rowId` (required): row id (e.g. `"1"` or `"1.0"`)
    - `offset` (optional): offset of the first item to return (default 0)
    - `limit` (optional): maximum number of items to return (default 100, max 1000)
  - **Response**: JSON array of all items for the given sheet and rowId.


