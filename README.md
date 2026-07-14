# TechToss Application

**Authors:** Alexander Liu / Julian Kroeger-Miller

**Status:** Planning & design

## Overview

A suite of internal tools to document and tabulate TechToss's electronics inventory and the donations and sales made to clients. TechToss is a nonprofit, so accurate dollar-value tracking of donated items is a first-class requirement — it feeds donor tax receipts and the totals reported to the board and funders. The broader goal is to replace ad-hoc spreadsheets with tools any staff member can search, filter, and update easily, while keeping an accurate record of what we hold and the value of what we give away and sell.

The system is made up of three parts:

1. **Inventory** — the single source of truth for everything we physically hold.
2. **Donations & Transactions** — the record of items leaving TechToss, whether donated or sold, and their dollar value.
3. **Portal** — the staff-only front door that ties the apps together and hosts shared login.

## Objectives

- Establish the Inventory application.
- Establish the Donation/Transaction application.
- Establish the TechToss portal.
- Design a shared authentication layer once, up front, that all three apps use.

## Inventory

The single source of truth for what TechToss holds: what an item is, where it is, its condition, and whether it's available to donate or sell. Staff should be able to search and filter this far more easily than a spreadsheet allows.

The app handles two kinds of records:

- **Unique items** — a single item with its own identity (e.g., a specific laptop with a serial number). Carries full specs: OS, RAM, storage, manufacturer, model, CPU, etc.
- **Bulk records** — a count of genuinely identical units (e.g., 40 USB-C cables). Each bulk record is homogeneous — the specs describe every unit in it. Items that differ in kind get separate records (USB-C cables and micro-USB cables are two records, not one).

Every record carries a tracking mode (unique vs. bulk), category, status, location, condition, source/donor, date received, and notes. Because a cable and a laptop have very different specs, technical details are handled as a flexible attribute set rather than a fixed set of columns.

**Key features**
- Item intake for single items (full specs) or bulk stock (by quantity).
- Search and filtering by category, status, location, and spec fields — the main reason to build this over a spreadsheet.
- A status lifecycle (e.g., received → testing → refurbished/ready → reserved → transferred/out) so staff can see at a glance what's available.
- Location management: move items between locations, view everything at a location.
- Stock reporting and an "available to donate/sell" view that feeds the Transaction app.

**Open question:** whether condition is tracked at the bulk-record level (e.g., "new USB-C cables" vs. "used USB-C cables" as separate records) still needs to be decided.

## Donations & Transactions

Records every item that leaves TechToss — whether **donated** (given away, tracked for its dollar value) or **sold** (revenue-generating) — along with the recipient, the value, and the date. Donations and sales share most of their structure, so they're modeled as one transaction type with a flag rather than two separate systems.

**Core records**
- **Transaction** — one per donation or sale event: type, date, client/recipient, staff member, total value, status, notes.
- **Line items** — the inventory items included in a transaction (linked inventory record, quantity, per-unit valuation, condition). This is the link back to Inventory.
- **Client/Recipient** — who received the items (individual, school, nonprofit, buyer), with contact info and history; needed for donation receipts.
- **Valuation** — the assessed dollar value of an item, especially for donations reported at fair market value.

**Key features**
- Record a transaction by selecting a type and client and adding inventory items; completing it updates stock in the Inventory app (decrement bulk counts, flag unique items as transferred).
- **Donor tax receipts** — generate a printable/exportable receipt per recipient showing the fair market value of donated items. As a nonprofit this is a primary output, not a nice-to-have, so donations should capture everything a valid receipt needs (recipient details, item descriptions, assessed values, dates).
- Reporting: total dollars donated over a period (for board and funder reporting), sales revenue, and item movement history.

**Cost/value analysis (API):** an external pricing API (e.g., eBay sold-listings or a refurbished-electronics source) can suggest or validate values — sale prices for sold items, fair market value for donations. Recommended approach: query on-demand and cache results, and always allow a manual override, since used-electronics pricing is noisy and the API will miss older or obscure items.

## Portal

A staff-only hub and connective tissue for TechToss's internal tools. It owns no domain data of its own — it provides the single login, the navigation, and an overview the individual apps can't give alone.

The portal is built to grow. Inventory and Transactions are the first two apps to live under it, but it should treat "an app" as a general slot so future tools (reporting, refurbishment tracking, volunteer scheduling, etc.) can be added to improve workflow and tabulation without redesigning the whole system.

**Responsibilities**
- Authentication and single sign-on — one login granting access to every app by role.
- App launcher / navigation — an extensible home listing the apps a staff member can access.
- Dashboard — key numbers pulled from whichever apps are installed (stock counts and items ready to move; recent transactions and dollars donated this period).
- Central user and role management.

**Extensibility:** each app is treated as a plug-in module the portal knows about — name, link, allowed roles, and an optional dashboard widget. Adding a future app becomes "register a new module." For a project starting with two apps, a simple hand-maintained list of apps and widgets is the right first step, with a more formal module contract added only if the number of apps grows.

## Security

Security is a shared concern across all three apps, and the portal is the most critical piece because it's the front door — compromise it and you're into every app beneath it.

- **Authentication** lives at the portal: password hashing, session handling, and (worth considering) two-factor, using an established framework rather than anything hand-rolled. No public sign-up — access is limited to TechToss staff accounts.
- **Role-based access control** is defined centrally and enforced in each app. Example: staff can record transactions, but only admins can void them or edit valuations.
- **Audit logs** for money-handling actions (who created/changed/voided a transaction) and for logins, plus a change log on inventory records.

## Open Questions

- Tech stack for the three apps (not yet decided).
- Whether condition is tracked at the bulk-record level in Inventory.
