# coinwarrior

Local-first CLI tool for tracking personal finances.

## Current Features

- Transaction entry with flags and interactive prompts for missing fields: `coinw add`
- Future-value calculator: `coinw future <value>`
- Interactive CSV import: `coinw import <file.csv>`
- Duplicate warning on `add` and `import` when a transaction matches an existing
  one's date, category, amount, and type
- Interactive transaction editing: `coinw edit` (category is chosen from a list)
- Interactive transaction deletion: `coinw delete`
- Interactive account management: `coinw account`
- Category management: `coinw category list|add|rename|delete`
- Monthly overall and category budgets with independent rollover: `coinw budget`
- Recurring transactions with flags and TUI fallback: `coinw recurring list|add|edit|delete|generate`
- Account balances report: `coinw report account`
- Range overview report: `coinw report overview [category] <range>`
- Transactions report: `coinw report transactions [category] <range>`

## Installation

### Pre-built binary (recommended)

Download the latest release for your platform from the [Releases page](https://github.com/amiraminb/coinwarrior/releases), extract, and place `coinw` somewhere on your `PATH`.

One-liner for macOS/Linux (replace `VERSION` with the latest tag):

```bash
VERSION=0.1.0
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/aarch64/arm64/;s/x86_64/x86_64/')
curl -sSL "https://github.com/amiraminb/coinwarrior/releases/download/v${VERSION}/coinwarrior_${VERSION}_${OS}_${ARCH}.tar.gz" \
  | tar xz -C /tmp \
  && mv /tmp/coinw ~/.local/bin/coinw
```

### From source with Go

```bash
go install github.com/amiraminb/coinwarrior@latest
```

### From source with make

Builds and copies `coinw` to `~/.local/bin`:

```bash
make build
```

If `~/.local/bin` is not on your `PATH`, either add it or run with the full path: `~/.local/bin/coinw <command>`.

## Quick Start

- Initialize data files:

```bash
coinw init
```

- Add at least one account:

```bash
coinw account
```

- Add a transaction:

```bash
coinw add
```

Supply any fields as flags to skip their prompts:

```bash
coinw add -t expense -a 20 -d 2026-05-21 -c Housing
coinw add -t expense -a 20 -d 2026-05-21
```

The second command opens the category picker first, then asks for an account and
an optional note. Any omitted field is collected in the TUI, including type,
amount, and date. Running `coinw add` with no flags opens the full form.

To save without the entry form, supply every field (`--note ""` means no note):

```bash
coinw add -t expense -a 20 -d 2026-05-21 -c Housing --account Checking --note ""
```

Flags: `-t`/`--type` (`expense` or `income`), `-a`/`--amount`, `-d`/`--date`
(`YYYY-MM-DD`), `-c`/`--category`, `--account`, and `-n`/`--note`. Category and
account flags match saved names case-insensitively; omit the flag to create a new
one through the TUI. Amounts use CAD, as in the interactive add form. Invalid flag
values fail before prompting or saving.

If the transaction you are entering matches an existing one on date, category,
amount, and type, `add` asks for confirmation before saving so an accidental
double entry is caught. `import` does the same on each row, including against
rows saved earlier in the same run. Transfers are never flagged, since they all
share the `Transfer` category and legitimately repeat. Currency is not part of
the match, so `20.00 CAD` and `20.00 USD` on the same date and category both
trigger the warning.

After saving, `coinw add` shows the amount's projected value after 5, 10, 15,
and 20 years at 5% annual compound interest.

- Calculate the future value of an amount:

```bash
coinw future 100.00
```

This shows the projected value after 5, 10, 15, and 20 years at 5% annual
compound interest.

- List transactions in a range:

```bash
coinw report transactions <range>
```

- List transactions for a single category, with an income/expense summary:

```bash
coinw report transactions <category> <range>
```

The category is matched case-insensitively against your saved categories, so
`groceries` and `Groceries` both work. An unknown name fails with the list of
valid categories rather than printing an empty table. Transfers are excluded
from a category-filtered listing, matching `coinw report overview`, so the table
and the summary below it always agree.

The report also shows total spending projected at 5% annual compound interest
after 5, 10, 15, and 20 years, grouped by currency. Income and transfers are not
included in spending projections.

- Generate an overview for a range (per-category totals, an income/expense
  summary, and the month's budget when the range is exactly one calendar month):

```bash
coinw report overview <range>
```

A range spanning more than one month also gets a per-month income/expense bar
chart. Bars scale to the largest value in the range, and a month the range only
partly covers is labelled with the days it covers.

The chart and per-month breakdown omit months with no income or expense
transactions matching the report. Each currency's chart shows only its active
months, even when the range extends far into the future.

The overview includes a range-level spending projection at 5% annual compound
interest for 5, 10, 15, and 20 years, grouped by currency.

```
Monthly Income / Expense (CAD)

 MONTH              INCOME                        EXPENSE
 2026-01 (15-31)    ███████████████████ 5,000.00  ████████████ 3,240.50
 2026-02            ███████████████████ 5,000.00  ████████████████████████ 6,100.00
 2026-03 (01-20)    █████████ 2,500.00            ███████ 1,890.00
```

The Per-Month Breakdown section is hidden by default. Add `--verbose` (or `-v`)
to show each month's category totals and income/expense summary below the chart:

```bash
coinw report overview year --verbose
coinw report overview Groceries year -v
```

A partial month counts only the days the range covers, and the `% EXP` share is
recomputed within each month rather than carried over from the whole range.

Without a category filter, every saved category is listed, including the ones
with no activity, so each month has the same set of rows and a category you spent
nothing on is visible rather than missing.

Each per-month table also compares itself to the month immediately before it. Red
`▲` means the month moved for the worse (more spent, or less earned) and green `▼`
means it improved. A category with no figure last month is compared against zero:

```
 CATEGORY              CUR    TOTAL      TXNS   % EXP     VS PREV
 Groceries             CAD    -150.00    1      100.0%    ▲ 50.00
 Income                CAD    5,500.00   1      0.0%      ▼ 1,000.00
 Dining                CAD    0.00       0      -         =
```

Pass a category to narrow the range totals, monthly bars, and verbose breakdown
to just that category. Other categories' zero-value rows are omitted:

```bash
coinw report overview <category> <range>
```

- Show account balances:

```bash
coinw report account
```

## Recurring Transactions

`coinw recurring` opens the action menu. Each action can also be run directly:

```bash
coinw recurring list                # includes notes
coinw recurring -l                  # also --list
coinw recurring add                 # full TUI form
coinw recurring edit                # choose a rule, then edit it
coinw recurring delete              # choose a rule and confirm deletion
coinw recurring generate            # generate through the current month
```

Supply flags to add a rule directly, or prompt only for missing required fields:

```bash
coinw recurring add -t expense -a 14.16 -c Fitness --account Checking --day 5 --note "Carbon Diet"
coinw recurring add -t transfer -a 100 --account Checking --to-account Savings --day 15
coinw recurring edit rec_123 --amount 19.99 --note "New price"
coinw recurring edit rec_123 --end-date "" --note ""
coinw recurring delete rec_123
```

Add/edit flags: `-t`/`--type`, `-a`/`--amount`, `--currency`, `-c`/`--category`,
`--account`, `--to-account`, `-d`/`--day` (1–28), `--start-date`, `--end-date`,
and `-n`/`--note`. Dates use `YYYY-MM-DD`; category and account names match saved
names case-insensitively. Flag-based adds require type, amount, category, account,
and day; transfers use a destination account instead of a category. Optional
fields default to CAD, today's start date, no end date, and no note.

Edits with flags update only supplied fields. Omit the ID to pick a rule, or
omit the flags to open its edit form. Deleting by ID is immediate and keeps
previously generated transactions.

Every command warns on stderr when recurring transactions have not been generated
through the current month. This check only reads the schedule. Run
`coinw recurring generate` to catch up; repeating generation does not duplicate
transactions. A month's occurrences are due for generation as soon as that month
begins, even when their scheduled day is later in the month.

## Managing Categories

```bash
coinw category list
coinw category add "Food Delivery"
coinw category rename "Food Delivery" "Meal Delivery"
coinw category delete "Unused Category"
```

`coinw category` also lists categories, and `coinw categories` is an alias.
Names match case-insensitively; quote names containing spaces. `edit` is an
alias for `rename`. Renaming updates matching transactions, recurring rules,
and category budgets. Deletion requires that the category has no references;
reassign its transactions, recurring rules, and budgets first. An existing
category name cannot be reused by `add` or by renaming another category.

## Category Budgets

Run `coinw budget` to set a monthly budget interactively. Choose **All spending**
for an overall limit, or a saved category for an individual limit. You can also
set and view budgets directly:

```bash
coinw budget set Groceries 2026-11 500
coinw budget set Dining 2026-11 450
coinw budget set "Personal Care" 2026-11 150
coinw budget show 2026-11
```

Amounts are in CAD by default; use `--currency USD` for another currency.
Each month, currency, and category has its own budget. Setting it again updates
that limit. Existing budgets without a category remain overall limits, and
`coinw budget set 2026-11 4500` sets one directly. Overall and category limits are
independent views of the same spending, so they should not be added together.
A category without a budget has no assigned limit.

Use `--carryover` to carry the previous month's remaining amount for the same
category and currency, or `--carryover=false` to decline it. The interactive
form asks when a previous month is awaiting a decision. Both unused funds and
overspending can roll forward. For example:

```bash
coinw budget set Travel 2026-12 400 --carryover
```

Budget spending counts expense transactions; income and transfers are excluded.
Overview reports for a full calendar month show category budgets alongside any
overall limit. A category-filtered overview shows only that category's budget.

## Supported Report Ranges

- `today`
- `yesterday`
- `week`
- `lastweek`
- `month`
- `lastmonth`
- `year`
- `lastyear`
- `<YYYY-MM-DD..YYYY-MM-DD>`

Example:

```bash
coinw report transactions 2026-04-01..2026-04-30
coinw report transactions Groceries 2026-04-01..2026-04-30
coinw report overview 2026-04-01..2026-04-30
```
