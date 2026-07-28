# TechToss-Application

Authors: Alexander Liu / Julian Kroeger-Miller

## Overview

A suite of internal, staff-facing applications for **TechToss**, a nonprofit that
documents and tabulates its inventory of electronics/projects and the
donations/transactions it handles with clients.

Because TechToss is a nonprofit, **donor tax receipts and dollar-value reporting
are primary outputs, not optional extras** — this shapes design priorities
throughout, particularly in the Transactions app, where IRS documentation
considerations apply.

> **Status:** Planning & design. Nothing is built yet. Structured outlines exist
> for all three applications; the sections below capture the design decisions
> confirmed so far and the questions still open.

## Applications

The suite is composed of three applications:

- **Inventory** — tracks items using a unique-vs-bulk model
- **Donations/Transactions** — handles donations and sales as a unified transaction type
- **Portal** — an extensible hub for future staff tools

### Reminder

We still need to establish proper security measures across all three
applications, such as user accounts and passwords.

## Inventory

Tracks the details of both individual items and items held in bulk — including
location, status, and important specs (operating system, RAM, manufacturer,
etc.). This could be simplified with Excel, but the goal is documentation that's
easier for any user to access and filter through.

Design decisions confirmed so far:

- **Unique vs. bulk model.** Items are tracked either as unique records or as
  bulk records.
- **Bulk records must be homogeneous.** Different item types require separate
  records rather than sub-variants within a single record.
- **Flexible spec attributes per category**, so each category can carry the
  specs relevant to it.
- **Status lifecycle** for tracking an item's state over time.
- **Location management** for where items are held.

Open question:

- Whether **condition** is tracked at the bulk-record level.

## Donations & Transactions

Tracks items donated and sold to clients, with emphasis on the dollar value
donated via electronic items. This is where the nonprofit reporting requirements
(donor tax receipts, IRS documentation) concentrate.

Design decisions confirmed so far:

- **Unified transaction type.** Donations and sales share one transaction type,
  distinguished by a flag, rather than being modeled separately.
- **Line items link directly to Inventory records**, so stock accuracy is
  maintained automatically as transactions are recorded.
- **Fair-market valuation via API** for cost analysis of items, with on-demand
  querying, caching of results, and support for manual override.

## Portal

An extensible hub for future staff tools, framed as a **module registry
pattern** — new applications can be added with minimal friction rather than
being wired into a single fixed dashboard.

## Open Questions (project-wide)

- **Tech stack** — not yet decided.
- **Inventory** — whether condition is tracked at the bulk-record level.