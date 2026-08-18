# TechToss-Application

Authors: Alexander Liu / Julian Kroeger-Miller

## Overview

A suite of internal, staff-facing applications for **TechToss**, a nonprofit that
documents and tabulates its inventory of electronics/projects and the
donations/transactions it handles with clients.

Because TechToss is a nonprofit, **donor tax receipts and dollar-value reporting
are primary outputs, not optional extras** — this will shape design priorities
once the Donations/Transactions app starts, particularly around IRS
documentation considerations.

> **Status:** Active development is currently focused entirely on **Inventory**,
> which is built and in day-to-day use for tracking machines and parts.
> Donations/Transactions and Portal are still unstarted — the sections below for
> those two are original planning notes, not built functionality.

## Applications

The suite is planned as three applications, one of which is built so far:

- **Inventory** ✅ *built, in use* — tracks machines and their parts using a
  unique-vs-bulk model (see below for the full feature list)
- **Donations/Transactions** — planned; handles donations and sales as a
  unified transaction type. Not started.
- **Portal** — planned; an extensible hub for future staff tools. Not started.

### Reminder

Inventory currently gates editing by network (CIDR allowlist) rather than real
user accounts — see [Inventory's README](Inventory/README.md#options). Proper
accounts/passwords across the suite are still an open item, and will matter
more once Donations/Transactions (with its tax-receipt implications) is built.

## Inventory

Tracks the details of both individual machines and parts held in bulk —
location, workflow status, condition, and free-form specs — so staff can find
and update anything without digging through a spreadsheet. It's a single Go
binary backed by SQLite; see [Inventory/README.md](Inventory/README.md) for
build/run instructions, the full route table, and the JSON API.

### Core model

- **Machines** — a unique record per computer, with an auto-generated asset tag
  (`TT-0001`), moving through a workflow: `Intake → Testing → Repair →
  Refurbished → For Parts → Sold → Tossed`.
- **Parts** — bulk records (CPU, GPU, RAM, Storage, Motherboard, PSU, Cooling,
  Case, Cabling, Network, Optical, Peripheral, …), each carrying a quantity, a
  condition (`Working` / `Untested` / `Faulty`), and free-form spec text. A
  part is either installed in a machine or sitting in loose inventory.
- **Location tracking** — every machine and every loose part is placed at a
  facility and sub-location (e.g. "Techtoss / Rack A"), with location-grouped
  and category-grouped views of loose inventory.
- **Condition** is tracked at the part level (this resolves the open question
  from the original design notes).

### Machine & part workflows

- Log a machine and record its installed parts in one form, either pulling
  parts from existing loose inventory or entering them fresh.
- Create several identical machines at once via a quantity field on the
  new-machine form, instead of repeating the form N times.
- Pull a part from loose inventory onto a machine in a partial quantity (e.g.
  1 of 15 fans), splitting the remainder back into inventory.
- Return an installed part to loose inventory, choosing its destination
  location; if an identical line already exists there, it merges in rather
  than creating a duplicate.
- Multi-select machines (click, ctrl/cmd-click, shift-click range-select) and
  bulk **group-edit** any combination of fields across the selection —
  including tagging the whole selection with a shared **client/group tag**
  (e.g. "Smith Family Order") to track machines built for the same
  client/order — or bulk-move all their installed parts back to inventory.

### Finding things

- A dedicated Search page and an always-available topbar search, both with
  search-as-you-type, an idle view showing recently-added items, a
  machine/part-category scope filter, sortable result columns, and
  keyboard-driven navigation (arrow keys + Enter) through results.
- Workshop (machine list) sorting by any column and status filtering, and
  Inventory filtering by category tag, with a toggle between location- and
  category-grouped views.
- Notes on machines and parts are visible from list views via an expandable
  bubble, and long names/models truncate with a click-to-expand instead of
  breaking the layout.

### Other

- Keyboard shortcuts (`/` to search, `1`/`2`/`3` to jump between pages,
  `Shift`+`=` to log a new machine, and more) with a discoverable `?` help
  panel.
- A version badge in the topbar tracking the current git tag (`git describe`),
  so staff can tell which build they're looking at.
- TSV export of machines and loose parts, and a JSON API for integrations
  (label printers, bots, dashboards).
- A retro-sunset visual identity (see `Inventory/static/techtoss.css`) and
  seed data (`-seed`) for demoing/testing against a realistic-sized dataset.

## Donations & Transactions

*Not started. Original planning notes, kept for when work begins:*

Tracks items donated and sold to clients, with emphasis on the dollar value
donated via electronic items. This is where the nonprofit reporting requirements
(donor tax receipts, IRS documentation) concentrate.

Design decisions confirmed so far:

- **Unified transaction type.** Donations and sales share one transaction type,
  distinguished by a flag, rather than being modeled separately.
- **Line items link directly to Inventory records**, so stock accuracy is
  maintained automatically as transactions are recorded — Inventory's data
  model is already built and ready to be linked against.
- **Fair-market valuation via API** for cost analysis of items, with on-demand
  querying, caching of results, and support for manual override.

## Portal

*Not started. Original planning notes, kept for when work begins:*

An extensible hub for future staff tools, framed as a **module registry
pattern** — new applications can be added with minimal friction rather than
being wired into a single fixed dashboard.

## Open Questions (project-wide)

- **Tech stack** — decided for Inventory: Go (`net/http` + `html/template`,
  no framework) and SQLite. Still open for Donations/Transactions and Portal;
  building them as more Go services in the same style is the natural default
  but hasn't been confirmed.
- **Accounts/permissions** — Inventory uses network-based edit gating as an
  interim measure; real accounts across the suite are still undecided.
