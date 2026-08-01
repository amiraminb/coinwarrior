# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `coinw report transactions [category] <range>` now takes an optional category
  as the first argument, filtering the listing to that category and printing an
  income/expense/net summary per currency below the table. The category is
  matched case-insensitively against the saved categories; an unknown name is
  rejected with the list of valid categories. Transfers are excluded from a
  category-filtered listing, matching `report budget`.
- Shell completion for `report transactions` suggests saved categories and the
  supported range keywords.

### Changed

- The income/expense summary table is now shared between `report budget` and
  `report transactions` instead of being implemented separately in each.

## [0.2.0] - 2026-07-04

### Added

- Recurring transactions with idempotent monthly generation: `coinw recurring`
  (list, add, edit, delete rules, and generate due transactions). The current
  month's occurrence is generated as soon as the month begins, even when the
  rule's day-of-month is later in the month, and is dated to that scheduled day.
- `report` is now a parent command with subcommands: `coinw report account`,
  `coinw report budget <range>`, and `coinw report transactions <range>`.

### Changed

- The former `coinw report <range>` is now `coinw report budget <range>`, and
  the former `coinw list <range>` is now `coinw report transactions <range>`
  (the range is now required).

### Removed

- Top-level `coinw list` command (moved under `coinw report transactions`).
- The `--details` flag and the "Transactions By Category" section from the
  budget report.

## [0.1.0] - 2026-05-16

### Added

- Interactive transaction entry: `coinw add`
- Interactive transaction editing: `coinw edit`
- Interactive transaction deletion: `coinw delete`
- Interactive account management: `coinw account`
- Monthly budgets with rollover: `coinw budget`
- List transactions: `coinw list [range]`
- Range report with category breakdown and balances: `coinw report <range> [--details]`
- Initial data setup: `coinw init`

[0.2.0]: https://github.com/amiraminb/coinwarrior/releases/tag/v0.2.0
[0.1.0]: https://github.com/amiraminb/coinwarrior/releases/tag/v0.1.0
