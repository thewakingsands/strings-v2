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

### Running several instances

A serving instance opens its index read only and never writes to it, so the index
takes a shared lock and any number of instances can serve from one `data` directory
at the same time:

```bash
go run . -addr ":8080" -data /srv/xivstrings &
go run . -addr ":8081" -data /srv/xivstrings &
```

Two things to know when scaling out:

- **The first index build needs one instance on its own.** Building writes, which
  takes an exclusive lock, so bring up a single instance until `data/index/<version>/`
  exists and only then start the rest. An instance that cannot get the lock gives up
  after 30 seconds and reports it rather than hanging.
- **An update only reaches the instance that ran it.** `POST /api/version` builds the
  new index and swaps the store on that one instance; the others keep serving the
  version they opened at startup. They pick up the new one when restarted, since
  `data/version` has already been rewritten. Building writes to
  `data/index/<new version>/`, a different directory from the one the others are
  reading, so it never disturbs them.

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
  - Query parameters: `lang` required, `q` required, `sheet` optional, `fields` optional, `offset` optional, `limit` optional, `mode` optional
  - Response: JSON with matching items and meta fields such as `total` and `elapsed`
  - See [Searching several languages at once](#searching-several-languages-at-once),
    [Advanced queries](#advanced-queries) and [Values and highlights](#values-and-highlights)

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

### Advanced queries

`mode=advanced` reads `q` as a [bleve query string](https://blevesearch.com/docs/Query-String-Query/)
instead of plain words:

```bash
curl -G "http://127.0.0.1:8080/api/search" --data-urlencode "lang=chs,en" \
  --data-urlencode "mode=advanced" --data-urlencode "q=*鲈* -sheet:Addon"
```

- A clause without a field, such as `*鲈*` or `-onion`, is searched in every `lang`,
  weighted by their order like a plain search. `+a +b` needs each of them in some
  language, not both in the same one, and `-a` drops a row that has `a` in any of them.
- A clause with a field, such as `en:onion` or `sheet:Item`, searches only that field,
  which may be a language outside `lang`.
- Wildcards and regular expressions match single index terms. The CJK columns are
  indexed as pairs of characters, which is why the plain search cannot find a single
  character like `鲈` and `*鲈*` can; it is also why its highlight covers the pairs
  `*鲈*` matched, not the character alone.
- A query that does not parse is rejected with `400 invalid query: ...`, and an unknown
  `mode` with `400 invalid mode: ...`. Leaving `mode` out, or `mode=simple`, is the plain
  search.

### Values and highlights

`values` holds the complete, raw value of each requested language column, and
`highlights` holds a short snippet of each one:

```json
{
  "sheet": "Item",
  "rowId": "4785",
  "values": {
    "en": "wizard eggplant\nwizard eggplants\nA firm purple vegetable.\nWizard Eggplant",
    "chs": "巫师茄子"
  },
  "highlights": {
    "en": "wizard <mark>eggplant</mark>\nwizard eggplants\nA firm purple…",
    "chs": "巫师茄子"
  },
  "index": 12
}
```

- `values` is the untouched game text. Nothing is escaped and nothing is truncated.
- `highlights` is HTML escaped (`&amp;` `&lt;` `&gt;` `&#34;` `&#39;`), wraps matches in
  `<mark>`, and is cut to roughly 200 characters around the best match with `…` marking
  where text was dropped. Render it as HTML or unescape it before display.
- Every language in `fields` gets a snippet, not only the ones searched. A language the
  query matched has its matches wrapped in `<mark>`; one it did not match is simply the
  start of the value. Both are cut by the same fragmenter, so all the columns of a row
  come back a comparable length rather than one snippet beside thousands of characters.
- Matching for a language outside `lang` runs that language's own analyzer, so it agrees
  with the index: quotes are separators, English is stemmed, and CJK is cut into bigrams.
  A query typed with `"Software"` therefore highlights a value stored with `“Software”`.
- `GET /api/items` does no matching and returns no `highlights` at all.

Before this split, `values` carried the truncated snippet for the searched language,
which made the full text unavailable to clients. Clients that relied on that snippet
should read `highlights` and fall back to `values`.
