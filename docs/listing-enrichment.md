# Listing enrichment: decisions and review

## Keep the existing architecture

CRUD still follows router → handler → CarService → sqlc → PostgreSQL. Mutable
cars are read directly from PostgreSQL. The runtime model remains sqlc's `db.Car`; adding a second domain
Car and a mapping layer would not improve this small service. Swagger response DTOs
were updated, and real HTTP tests verify their compatibility with `db.Car` JSON.

Suggestions follow router → GenerationHandler → GenerationService → GenerationSource.
The source interface has one method. `provider.GenerationCatalog` implements it
using embedded JSON, loaded and checked once at startup. The shared catalogue is
immutable; callers get fresh slices. The service applies inclusive year matching
and deterministic ordering. Main injects the source explicitly. No package fetches
remote data at runtime; source URLs are evidence links, not HTTP integrations.

CRUD never calls suggestions. Unknown cars and owner-written generation labels
remain valid inputs to the existing CRUD rules.

## Optional means three different update states

| Input | POST | PUT |
|---|---|---|
| Field omitted | SQL NULL | Keep stored value |
| `null`, `""`, whitespace only | SQL NULL | Clear to SQL NULL |
| Nonblank string | Trim edges and save | Trim edges and replace |

All responses include string-or-null keys. Description line breaks are retained.
The JSON DTO's small `OptionalText` decoder tracks presence separately from value.
A plain `*string` would lose the distinction between omission and explicit null.
sqlc generates nullable `pgtype.Text` values and update-presence booleans. The UPDATE
uses CASE expressions in one statement; there is no read/merge/write race.
Core fields keep their existing PUT replacement semantics. Changing make/model/year
does not automatically erase an omitted generation: the caller should send null if
it needs to clear a previously selected value.

Example create (a user supplies the URL; the example is not a discovered image):

```json
{
  "brand": "Mercedes-Benz",
  "model": "E280",
  "production_year": 1995,
  "color": "Black",
  "price": 9000,
  "image": "https://your-media-host.example/cars/my-car.jpg",
  "model_generation": "W124 facelift",
  "description": "Service records available.\nInspection welcome."
}
```

## Image reference, not image storage

- Binary PostgreSQL data would increase database and backup size.
- A filesystem path would depend on one host/container and need a media server.
- An S3 key is useful only when an upload/storage service actually owns the object.
- A URL meets today's manual input needs and permits a future stable media domain.

HTTP is allowed for local development; use HTTPS in production. Credentials,
fragments, whitespace, non-HTTP schemes and oversized URLs are rejected. There is
no server-side download, so this is not an arbitrary-URL fetch/SSRF endpoint. The
URL's existence, MIME type, ownership and content are not verified. Rendering a
third-party URL can expose the viewer to its host; frontends need a deliberate
image-host policy. Descriptions are untrusted plain text, not trusted HTML.

Later: add authenticated uploads, size/type checks, and a media record containing
an opaque object key. Store bytes in S3, return a stable application/CDN URL, and
keep temporary signed URLs out of persistent car records. Private-media ownership
can use an added media ID while preserving the outward `image` URL contract.
No bucket, upload endpoint, filesystem convention or AWS dependency is added now.

## Generation contract and limits

`GET /cars/generations?brand=Mercedes-Benz&model=E280&year=1995`

```json
{
  "candidates": [
    {
      "generation": "W124 facelift",
      "production_year_start": 1993,
      "production_year_end": 1995,
      "body_style": "saloon",
      "source_url": "https://mercedes-benz-publicarchive.com/marsClassic/en/instance/ko/E-280--W-124-E-28-1993---1995.xhtml?oid=5158"
    },
    {
      "generation": "W210 early",
      "production_year_start": 1995,
      "production_year_end": 1999,
      "body_style": "saloon",
      "source_url": "https://mercedes-benz-publicarchive.com/marsClassic/en/instance/ko/210-series-E-Class-Saloons-1995---1999.xhtml?oid=5309"
    }
  ],
  "notice": "Suggestions only. Catalogue coverage is limited; no match does not mean an invalid car. Years are inclusive calendar years, not exact build dates or market-specific model years. Choose the generation yourself."
}
```

Zero, one and multiple candidates are all HTTP 200. Missing, repeated or invalid
required parameters return 400 `{"error":"..."}`. A source failure returns 503;
it is not disguised as an empty result. Context is propagated with a two-second
deadline; any future network source must honor it and use bounded HTTP clients.
Matching ignores case, whitespace and hyphens, but is not fuzzy: E280 CDI does not
match E280. There is no confidence score because this dataset cannot justify one.

The bundled catalogue deliberately contains only two E280 saloon entries. It does
not cover every E280 (or all markets/body styles), let alone every vehicle. This
is a small maintainable start, not an authoritative vehicle database. An external
API or imported dataset can replace the source without changing handlers or CRUD.

### Provenance and adding coverage

Reviewed 2026-09-20:

- [Mercedes-Benz E280 W124](https://mercedes-benz-publicarchive.com/marsClassic/en/instance/ko/E-280--W-124-E-28-1993---1995.xhtml?oid=5158): July 1993–August 1995.
- [Mercedes-Benz early W210 series](https://mercedes-benz-publicarchive.com/marsClassic/en/instance/ko/210-series-E-Class-Saloons-1995---1999.xhtml?oid=5309): initial body generation, including the 1997 engine change. The label “W210 early” is our presentation label, not an official trim designation.
- [E280 inline-six](https://mercedes-benz-publicarchive.com/marsClassic/en/instance/ko/E-280--W-210-E-28-1995---1997.xhtml?oid=5320) and [E280 V6](https://mercedes-benz-publicarchive.com/marsClassic/en/instance/ko/E-280-V6-engine--W-210-E-28-1997---1999.xhtml?oid=5310) document the two engine periods grouped under that body generation.

Calendar-year overlap does not claim the two cars shared an exact build month.
Before adding a row to `internal/provider/generations.json`, verify its make/model,
body style and dates against a credible source, retain the link here/in the row,
and add boundary/overlap tests. Do not copy a whole third-party database without
checking its licensing. Dataset updates require rebuild/redeploy, intentionally.

## Review findings and corrections

1. Omission and null are different. Preserving them avoids legacy clients erasing
   new metadata. Integration tests cover updates from clients unaware of the fields.
   Go DTO encoding also preserves this distinction using `omitzero` and `IsZero`;
   a round-trip test prevents omitted fields becoming null when marshalled.
2. COALESCE would preserve old values but make clearing impossible. Presence flags
   and atomic CASE updates support both behaviors without application-side merging.
3. An earlier Redis cache could return old listing data if PostgreSQL committed
   while cache invalidation failed. A regression test reproduced this. Reads now
   use PostgreSQL; update and delete need no cross-system invalidation.
4. Free-form text needs limits. HTTP requests are capped at 128 KiB, text lengths
   are checked before SQL, and database CHECKs protect alternate writers too.
   NUL text is rejected before PostgreSQL. Extra JSON documents are rejected.
5. Suggestions need uncertainty, not hidden defaults. Empty results stay arrays;
   overlaps remain multiple candidates; labels never become mandatory validation.
6. A variadic router dependency would silently accept multiple services. The final
   constructor uses one explicit service argument; tests pass nil when not needed.
7. Existing CI selected only concurrency/load tests. It now also runs the complete
   integration-tag suite, including optional-field and migration tests.
8. Migration tests apply old schema → insert old car → upgrade → save metadata →
   downgrade → upgrade, and verify surviving core data and length constraints.
   They run in a rolled-back private schema on the dedicated test database.

## Deployment and next milestone

Apply migration 000002 before starting the new binary. It adds nullable TEXT with
length checks and no data backfill. PostgreSQL still needs an ALTER TABLE lock:
schedule the migration appropriately and avoid long transactions during rollout.
Down migration deletes the three fields' contents; rollback is not a backup.
Drain old cached app instances before bringing up this version. Old Redis data
is not read by the new binary; do not delete a Redis volume without a separate
data-retention decision.
The local rollback Make target checks that the current clean version is 2, so a
second invocation cannot accidentally drop the original cars table.

Exact money input is required on POST and PUT and converted from the original
JSON number to integer cents before PostgreSQL. This prevents omitted prices
becoming zero and float64 rounding away a fractional cent.
Before public image uploads, authentication/ownership and media validation are
required. A bigger generation dataset should follow actual coverage needs, not
invented confidence or more abstraction.
