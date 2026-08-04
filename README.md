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

If that release check fails, from a network outage or a GitHub rate limit, the
server logs the failure and starts anyway on the version named in `data/version`,
serving that data unchanged. It only refuses to start when there is nothing local
to fall back to, such as a first deployment with no connectivity. Nothing is
retried in the background, so a degraded start keeps serving the older data until
someone triggers `POST /api/version`.

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
  - Query parameters: `lang` required, `q` required, `sheet` optional, `fields` optional, `offset` optional, `limit` optional
  - Response: JSON with matching items and meta fields such as `total` and `elapsed`
  - See [Searching several languages at once](#searching-several-languages-at-once) and
    [Values and highlights](#values-and-highlights)

- Get items by sheet
  - Endpoint: `GET /api/items`
  - Query parameters: `sheet` required, `offset` optional, `limit` optional
  - Response: JSON with items for the sheet and meta

- Version
  - `GET /api/version`: current data version and update status
  - `POST /api/version?token=...`: start async update from the latest ixion release
  - See [docs/update-api.md](docs/update-api.md) for the full contract

### Searching several languages at once

`lang` takes a comma separated list of language codes, and every one of them is
searched in a single request:

```bash
curl "http://127.0.0.1:8080/api/search?lang=chs,en,ja&sheet=Item&q=onion"
```

- The order is the preference order. `lang=chs,en,ja` ranks a row that matched in `chs`
  above one that matched only in `en`, which in turn ranks above one that matched only
  in `ja`, the same as boosting `chs^3 en^2 ja^1`. A single language is scored exactly
  as it was before this parameter accepted lists.
- Blank segments and repeats are ignored, so `lang=chs,,en` and `lang=chs,chs,en` both
  mean `lang=chs,en`.
- An unknown code is rejected with `400 invalid lang: <code>`, matching how `fields`
  already reports unknown languages. Earlier versions accepted anything and quietly
  returned no results.
- `meta.total` counts every matching row once, however many languages it matched in.

Supported codes are `chs`, `tc`, `en`, `de`, `fr`, `ja`, `ko`.

`fields` chooses which language columns come back and defaults to `chs,tc,en,ja`. It is
independent of `lang`: searching a language that `fields` leaves out is allowed, and
that language is simply absent from the response.

### Values and highlights

`values` holds the complete, raw value of each requested language column.
`highlights` holds the matching snippet for each language the search actually hit:

```json
{
  "sheet": "Item",
  "rowId": "4785",
  "values": {
    "en": "wizard eggplant\nwizard eggplants\nA firm purple vegetable.\nWizard Eggplant",
    "chs": "巫师茄子"
  },
  "highlights": {
    "en": "wizard <mark>eggplant</mark>\nwizard eggplants\nA firm purple…"
  },
  "index": 12
}
```

- `values` is the untouched game text. Nothing is escaped and nothing is truncated.
- `highlights` is HTML escaped (`&amp;` `&lt;` `&gt;` `&#34;` `&#39;`), wraps matches in
  `<mark>`, and is cut to roughly 200 characters around the best match with `…` marking
  where text was dropped. Render it as HTML or unescape it before display.
- A language only appears in `highlights` if the search matched it, so `highlights` is
  absent from responses of `GET /api/items`, and absent from a row that matched in
  another language. Its keys never go beyond what `fields` requested.

Before this split, `values` carried the truncated snippet for the searched language,
which made the full text unavailable to clients. Clients that relied on that snippet
should read `highlights` and fall back to `values`.
