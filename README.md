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

`-no-update-check` starts on the version named in `data/version` without asking
GitHub for a newer release, and refuses to start when there is no local data.

### Data directory layout

The `-data` path is the root for all app data. The server expects or creates:

- `data/version` - text file with the current version, for example `publish-20260303-8b409c8`
- `data/strings/<version>/` - extracted JSON files from `strings.zip`
- `data/index/<version>/` - Bleve search index for that version

On first run, or when the version changes, the server fetches the latest
[ixion release](https://github.com/thewakingsands/ixion/releases/latest),
downloads `strings.zip`, extracts to `data/strings/<version>/`, and builds the
index under `data/index/<version>/`.

An index keeps the mapping it was built with, which says how each language is
analyzed, and every query is answered with that copy rather than the server's own.
So a server whose mapping differs from its index's refuses to start, and lists the
fields that differ; otherwise a changed analyzer would quietly do nothing. Rebuild
the index with the new binary, or keep serving it with the old one. The startup log
prints the fingerprint it matched (`index mapping … matches this binary`), which
also tells two instances' indexes apart.

If that release check fails, from a network outage or a GitHub rate limit, the
server logs the failure and starts anyway on the version named in `data/version`,
serving that data unchanged. It only refuses to start when there is nothing local
to fall back to, such as a first deployment with no connectivity. Nothing is
retried in the background, so a degraded start keeps serving the older data until
someone triggers `POST /api/version`.

When the data directory is read only, or the version is pinned on purpose, pass
`-no-update-check` to skip the check altogether. It affects startup only:
`POST /api/version` still updates when asked.

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
  - `sheet` is a comma separated list of sheet patterns; see [Filtering by sheet](#filtering-by-sheet)
  - See [Searching several languages at once](#searching-several-languages-at-once),
    [Advanced queries](#advanced-queries), [Limits](#limits) and [Values and highlights](#values-and-highlights)

- Get items by sheet
  - Endpoint: `GET /api/items`
  - Query parameters: `sheet` required (one exact sheet name, no patterns), `offset` optional, `limit` optional
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
  in `ja`, the same as boosting `chs^3 en^2 ja^1`. With a single language there is no
  order to apply, and the result is that column's own ranking.
- A column that can only match the query through text nobody translated counts for a
  twentieth, wherever it sits in the order: `chs`, `tc`, `ja` and `ko` for a query
  written only in Latin letters, and `en`, `de` and `fr` for one with no Latin letters
  at all. So `lang=chs,en&q=onion` ranks a row whose `en` column says onion above one
  whose `chs` column was left in English. Those rows still match, and searching that
  column alone ranks them as before.
- Blank segments and repeats are ignored, so `lang=chs,,en` and `lang=chs,chs,en` both
  mean `lang=chs,en`.
- An unknown code is rejected with `400 invalid lang: <code>`, matching how `fields`
  already reports unknown languages. Earlier versions accepted anything and quietly
  returned no results.
- `meta.total` counts every matching row once, however many languages it matched in.

Supported codes are `chs`, `tc`, `en`, `de`, `fr`, `ja`, `ko`.

`ja` matches hiragana and katakana interchangeably: `q=かたかな` finds a row that spells
it `カタカナ`, and the reverse. Each part of the query counts in full where it matches
the script it was typed in and for less where it matches only the other one, so the
spelling typed ranks first. `meta.total` grows accordingly. Folding leaves a query with
no kana in it as it was. The other languages are not folded.

English stop words such as `the`, `but` and `how` never match in `en`, whose analyzer
drops them, and a query written only in Latin letters skips them in `chs`, `tc`, `ja`
and `ko` too, where the only English is text nobody translated. So `q=But how` finds
nothing in any of those five, and `q=the onion` matches there as `onion` alone. A query
with any other script in it, such as `Aランク`, keeps every word, since the Latin letters
then belong to a name. `de` and `fr` drop their own languages' stop words, not English
ones: `but` is a French word.

`fields` chooses which language columns come back and defaults to `chs,tc,en,ja`. It is
independent of `lang`: searching a language that `fields` leaves out is allowed, and
that language is simply absent from the response.

### Filtering by sheet

`sheet` on `/api/search` takes a comma separated list of sheet patterns. A leading `-`
excludes the sheets a pattern matches instead of including them:

```bash
curl -G "http://127.0.0.1:8080/api/search" --data-urlencode "lang=ja" \
  --data-urlencode "q=覚えていろ" --data-urlencode "sheet=quest/*,cut_scene/*"
curl -G "http://127.0.0.1:8080/api/search" --data-urlencode "lang=en" \
  --data-urlencode "q=onion" --data-urlencode "sheet=-Item,-Action,-Addon"
```

- `*` matches any run of characters, `/` included, and `?` exactly one. A pattern
  without either is an exact sheet name, so `sheet=Item` works as it always has.
  Matching is case-sensitive: `sheet=item` finds nothing.
- A row is kept when it matches at least one inclusion (or there are none) and none of
  the exclusions, so `sheet=quest/*,-quest/000/*` is every quest sheet outside
  `quest/000/`, and a list of only exclusions starts from every row.
- Filtering never changes the ranking: the result is the unfiltered order with the
  other rows removed, and `meta.total` counts the rows kept.
- A pattern that matches no sheet returns no rows, not an error. A `-` with nothing
  after it is rejected with `400 invalid sheet: -`.
- It applies in both modes. In advanced mode the query's own `sheet:` clauses still
  work alongside it.

The web UI's search box takes the same filter as tokens, in either mode:
`filename:quest/*` includes, `-filename:Item,Action` excludes each listed pattern, and
every such token is taken out of the query and sent as `sheet`.

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
- In `ja`, a word or `ja:` clause matches either kana script as the plain search does.
  A quoted phrase such as `"かたかな"` matches the script exactly as typed, and so do
  fuzzy (`~`), wildcard and regular expression clauses.
- A query that does not parse is rejected with `400 invalid query: ...`, and an unknown
  `mode` with `400 invalid mode: ...`. Leaving `mode` out, or `mode=simple`, is the plain
  search.
- A wildcard, regular expression or fuzzy clause may match at most 8192 index terms in
  a field; see [Limits](#limits).

### Limits

Both endpoints stop a search that would otherwise hold a core for minutes:

- A search still running after 8 seconds is stopped and answered with
  `504 search took longer than 8s and was stopped`. The deadline sits under the
  server's 10 second write timeout, past which no answer could be sent at all.
- A clause may expand to at most 8192 index terms in each field it searches, and one
  query may combine at most 8192 different terms. A query past either is rejected with
  `422 query expands to more than 8192 terms in <field>; ...` or
  `422 query combines more than 8192 terms; ...`. That leaves room for every sheet
  pattern, a one-letter prefix such as `en:s*`, and any value in the corpus pasted
  whole, while a bare `*` or a CJK term with `~2`, which match most of a column, are
  refused instead of running for tens of seconds.
- A term the query repeats is searched once and weighs as much as all its repeats. A
  pasted paragraph repeats its commonest words and character pairs, whose lists of rows
  are the longest to read, and reading each only once is what lets every value in the
  corpus, pasted whole, finish inside the deadline.
- A search whose client disconnects is stopped, with nothing written back.

The `sheet` parameter is exempt from the term limit: it filters without scoring, and
bleve merges a pattern's sheets instead of counting them. A `sheet:` clause in an
advanced query is scored like any other clause, and is limited.

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
- In a plain search, a language outside `lang` is matched the way searching it would
  match, so it agrees with the index: quotes are separators, English is stemmed, CJK is
  cut into bigrams, and English stop words are skipped as described above. A query typed
  with `"Software"` therefore highlights a value stored with `“Software”`, and a `ja`
  column marks a word in either kana script, as the search would match it.
- An advanced query marks only what it searched. A language outside `lang` is marked
  where a clause names it, as `de:zwiebel` does, and is otherwise the start of its value.
- `GET /api/items` does no matching and returns no `highlights` at all.

Before this split, `values` carried the truncated snippet for the searched language,
which made the full text unavailable to clients. Clients that relied on that snippet
should read `highlights` and fall back to `values`.
