# TechToss

Inventory for computers being refurbished and the components inside them.

TechToss tracks two things and the relationship between them:

- **Machines** — a computer that's come through the shop. Each gets an
  auto-generated asset tag (`TT-0001`) and moves through a workflow status:
  `Intake → Testing → Repair → Refurbished → For Parts → Sold → Tossed`.
- **Parts** — a component (CPU, GPU, RAM, Storage, PSU, Motherboard, …). A part
  is either **installed in a machine** or sitting in **loose inventory** (e.g. a
  good GPU you harvested from a scrapped tower). Parts carry a condition:
  `Working`, `Untested`, or `Faulty`.

The typical loop: log an incoming machine → record what's inside it → as you
refurbish, harvest good parts into loose inventory or scrap the rest → find
anything fast with search.

It's a single Go binary backed by one SQLite file. No web framework — just the
standard library `net/http` and `html/template`, with an in-memory search index.
Adapted from Henner Zeller's [stuff-org](https://github.com/hzeller/stuff-org).

## Build & run

Requires Go 1.21+ and a C toolchain (the SQLite driver uses cgo).

```sh
make            # build ./techtoss
./techtoss      # serve on http://localhost:2000
```

First run creates `techtoss.db` automatically. To start with a few example
machines and parts so the UI isn't empty:

```sh
./techtoss -seed
```

Then open <http://localhost:2000/>.

## Options

```
-bind-address string
      Listen address:port (default ":2000")
-dbfile string
      SQLite database file (default "techtoss.db")
-templatedir string
      Directory with HTML templates (default "./template")
-staticdir string
      Directory with static assets (default "./static")
-cache-templates
      Cache templates; set false for live template editing (default true)
-edit-permission-nets string
      Comma-separated CIDR networks allowed to edit (empty = everyone).
      Requests from outside these networks get a read-only view.
-seed
      Seed the database with example data if it is empty
```

Example: read-only for the world, editable from the shop LAN:

```sh
./techtoss -edit-permission-nets "192.168.1.0/24,10.0.0.0/8"
```

## Web UI

| Path                     | What it is                                   |
|--------------------------|----------------------------------------------|
| `/`                      | Workbench — machine cards, filter by status  |
| `/machine?id=N`          | A machine and its installed parts            |
| `/machine/edit?id=new`   | Log a new machine (`?id=N` to edit)          |
| `/inventory`             | Loose parts, grouped by category             |
| `/part/edit?id=new`      | Add a part (`?machine=N` to pre-assign)      |
| `/search`                | Search machines and parts (search-as-you-type)|

## JSON API

For integration with other tools (label printers, Slack bots, dashboards).

| Endpoint          | Query                        | Returns                          |
|-------------------|------------------------------|----------------------------------|
| `/api/search`     | `q`, optional `count` (100)  | Ranked machines + parts          |
| `/api/machine`    | `id`                         | A machine with its parts         |
| `/api/part`       | `id`                         | A single part                    |
| `/api/inventory`  | —                            | Loose (uninstalled) parts        |
| `/api/live`       | `q`                          | Flat rows for the type-ahead box |

### Example

```sh
curl "http://localhost:2000/api/search?q=rtx"
```

```json
{
  "query": "rtx",
  "count": 1,
  "results": [
    {
      "kind": "part",
      "part": {
        "id": 5,
        "machine_id": 2,
        "category": "GPU",
        "model": "NVIDIA RTX 2060",
        "spec": "6GB GDDR6",
        "quantity": 1,
        "condition": "Faulty"
      }
    }
  ]
}
```

## Docker

```sh
docker build -t techtoss .
docker run -p 2000:2000 -v "$PWD/data:/data" techtoss -dbfile /data/techtoss.db
```

## Layout

```
main.go              flags, wiring, HTTP server
store.go             Machine & Part models + Store interface
store-sqlite.go      SQLite implementation + schema
search.go            in-memory search index
handlers.go          web handlers (dashboard, detail, forms, inventory)
search-handler.go    search page + live type-ahead JSON
api.go               JSON API
templates.go         html/template renderer
seed.go              example data for -seed
template/*.html      server-rendered pages
static/techtoss.css  styling
```

## Notes

The search index lives in memory and is rebuilt from the database on startup,
then kept current on every write. It does case-insensitive, dash-insensitive,
AND-across-terms matching with light scoring (whole-word > prefix > substring) —
enough to make `16gb ddr4`, `faulty psu`, or a status name find the right rows.
