# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0] - 2026-11-05

### Added

- Type-to-filter support in all list menus, including categories, accounts,
  transactions, recurring rules, and actions. Matching is case-insensitive;
  arrows navigate matches, Enter selects, Backspace edits the filter, and Esc
  clears it before going back or cancelling.
- `coinw add` accepts type, amount, date, category, account, and note flags,
  prompting only for omitted fields. Supplying every field skips the entry form.
- Category management with `coinw category list|add|rename|delete`. Renaming
  updates matching transactions, recurring rules, and budgets; deletion requires
  the category to be unused. Categories are sorted alphabetically.
- Monthly category budgets alongside overall budgets, with independent rollover
  by category and currency. Set them with
  `coinw budget set <category> <month> <amount>` or the interactive form, and view
  them in budget and overview reports.
- `coinw future <value>` projects a one-time amount at 5% annual compound interest
  over 5, 10, 15, and 20 years. Transaction entry and spending reports also display
  future-value projections.
- Recurring rules can be added and edited with flags, with interactive prompts
  for missing required fields. Edit or delete rules directly by ID, and use
  `coinw recurring -l` or `--list` to list them.
- Recurring rule listings include notes, monthly income/expense/net totals per
  currency, and future-value projections. Commands warn when recurring
  transactions have not been generated through the current month.
- `coinw report transactions [category] <range>` now takes an optional category
  as the first argument, filtering the listing to that category and printing an
  income/expense/net summary per currency below the table. The category is
  matched case-insensitively against the saved categories; an unknown name is
  rejected with the list of valid categories. Transfers are excluded from a
  category-filtered listing, matching `report overview`.
- Shell completion for `report transactions` suggests saved categories and the
  supported range keywords.
- `coinw add` and `coinw import` now warn before saving a transaction that
  matches an existing one's date, category, amount, and type, asking for
  confirmation so an accidental double entry is caught. The import check also
  sees rows saved earlier in the same run. Currency is not part of the match, so
  the same amount in two currencies on one date and category is still flagged.
  Transfers are never flagged, since they all share the `Transfer` category and
  legitimately repeat.

- `coinw report overview <range>` now prints a per-month income/expense bar
  chart when the range spans more than one calendar month. Bars scale to the
  largest value in the range so income and expense stay comparable, a month the
  range only partly covers is labelled with the days it covers (for example
  `2026-01 (15-31)`), and transfers are excluded as in every other section.
- `coinw report overview <range> --verbose` (or `-v`) includes per-month category
  breakdown tables for multi-month ranges. A partial month counts only its
  covered days, and `% EXP` is recomputed within each month rather than carried
  over from the range. Monthly charts and breakdowns omit months with no matching
  income or expense transactions.
- `coinw report overview [category] <range>` now takes an optional category as
  the first argument. Every section, including the monthly bars and the per-month
  breakdown, then covers only that category. The name is matched
  case-insensitively against the saved categories, an unknown one is rejected
  with the list of valid names, and transfers are excluded.
- Each per-month category table carries a `VS PREV` column comparing that month
  to the one immediately before it: red `▲` when the month moved for the worse
  (more spent, or less earned) and green `▼` when it improved. A category with no
  figure in the previous month is compared against zero, so a new expense reads
  as worse and new income as better. The first month in a range has no baseline,
  so it has no such column.
- Every saved category now appears in each category table, including the ones
  with no activity, so a month has a fixed set of rows and an unspent category is
  visible rather than absent.

### Changed

- List menus now treat `q`, `j`, and `k` as filter text. Use Ctrl+C to quit and
  arrow keys to navigate; Yes/No confirmations retain their existing shortcuts.
- `coinw add` uses CAD without a separate currency prompt.
- `coinw edit` now picks the category from a list of saved categories instead of
  a free-text field, with a `[New category]` entry for adding one, matching how
  `coinw add` works. The cursor opens on the transaction's current category, so
  pressing enter leaves it unchanged. Transfers skip the category step entirely,
  since they always carry `Transfer` and never appear in a category breakdown.
- `coinw report budget <range>` is now `coinw report overview <range>`. The
  command prints per-category totals and an income/expense summary for any
  range, and the budget section only when the range is exactly one calendar
  month, so "budget" named the one section that is often absent and collided
  with the top-level `coinw budget` used to manage budgets.
- `coinw report` with an unknown subcommand now exits 1 with an error naming it,
  instead of printing help and exiting 0.
- The income/expense summary table is now shared between `report overview` and
  `report transactions` instead of being implemented separately in each.

### Fixed

- A CSV row whose amount the importer left unvalidated (for example `$45.67` or
  `1.234`, which the parser only flags for bad dates and debit/credit conflicts)
  aborted the whole import at save time, dropping every remaining row. The
  amount is now checked up front and reported on the row menu.

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

[Unreleased]: https://github.com/amiraminb/coinwarrior/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/amiraminb/coinwarrior/releases/tag/v0.3.0
[0.2.0]: https://github.com/amiraminb/coinwarrior/releases/tag/v0.2.0
[0.1.0]: https://github.com/amiraminb/coinwarrior/releases/tag/v0.1.0
