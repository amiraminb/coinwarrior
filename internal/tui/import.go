package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/amiraminb/coinwarrior/internal/importer"
	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/amiraminb/coinwarrior/internal/money"
)

const newAccountSentinel = "\x00new-account"

type rowAction int

const (
	rowSave rowAction = iota
	rowSkip
	rowQuit
)

// RunImport walks the parsed CSV rows one at a time, letting the user assign a
// category and save, skip, edit, or quit. Currency and account are asked once up
// front and applied to every saved row. Rows accepted before a quit stay saved.
func RunImport(rows []importer.ParsedRow) error {
	account, currency, ok, err := promptAccount()
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println("import cancelled")
		return nil
	}

	categories, err := Svc.LoadCategories()
	if err != nil {
		return err
	}

	saved, skipped := 0, 0
	for i, row := range rows {
		action, updated, category, err := runRow(row, i, len(rows), currency, account, categories)
		if err != nil {
			return err
		}
		switch action {
		case rowSave:
			if err := Svc.AddCategory(category); err != nil {
				return err
			}
			tx, err := Svc.AddTransaction(updated.Type, updated.AmountInput, currency, updated.Date, category, account, "", updated.Note)
			if err != nil {
				return err
			}
			fmt.Printf("saved transaction: %s\n", tx.ID)
			saved++
			if categories, err = Svc.LoadCategories(); err != nil {
				return err
			}
		case rowSkip:
			skipped++
		case rowQuit:
			remaining := len(rows) - i
			fmt.Printf("import stopped: %d saved, %d skipped, %d remaining\n", saved, skipped, remaining)
			return nil
		}
	}

	fmt.Printf("import complete: %d saved, %d skipped\n", saved, skipped)
	return nil
}

// runRow drives the interactive menu for a single row until the user saves,
// skips, or quits. A save attempt that fails validation (no category yet, an
// unresolved parse error, or a service rejection) keeps the user on the menu
// with the reason shown instead of aborting the whole import.
func runRow(row importer.ParsedRow, index, total int, currency, account string, categories []string) (rowAction, importer.ParsedRow, string, error) {
	category := ""
	warning := ""
	if row.ParseErr != nil {
		warning = row.ParseErr.Error()
	}

	for {
		title := HeaderStyle.Render(fmt.Sprintf("Row %d of %d", index+1, total))
		prompt := rowSummary(row, category, currency, account, warning)

		categoryLabel := "Set category"
		if category != "" {
			categoryLabel = "Change category (" + category + ")"
		}
		items := []selectionItem[string]{
			{label: categoryLabel, value: "category"},
			{label: "Save", value: "save"},
			{label: "Edit amount / date / note", value: "edit"},
			{label: "Skip", value: "skip"},
			{label: "Quit import", value: "quit"},
		}

		choice, chose, err := runSelection(title, prompt, items)
		if err != nil {
			return rowSkip, row, "", err
		}
		if !chose {
			// esc/q on the menu skips this row rather than quitting the run;
			// "Quit import" is an explicit menu item.
			return rowSkip, row, "", nil
		}

		switch choice {
		case "category":
			picked, ok, err := runCategoryPicker(categories)
			if err != nil {
				return rowSkip, row, "", err
			}
			if ok {
				category = picked
			}
		case "save":
			switch {
			case row.ParseErr != nil:
				warning = "fix the row before saving: " + row.ParseErr.Error()
			case category == "":
				warning = "choose a category before saving"
			default:
				return rowSave, row, category, nil
			}
		case "edit":
			updated, err := runRowEdit(row)
			if err != nil {
				return rowSkip, row, "", err
			}
			row = updated
			if row.ParseErr != nil {
				warning = row.ParseErr.Error()
			} else {
				warning = ""
			}
		case "skip":
			return rowSkip, row, "", nil
		case "quit":
			return rowQuit, row, "", nil
		}
	}
}

func rowSummary(row importer.ParsedRow, category, currency, account, warning string) string {
	s := renderField("Date: ", row.Date) + "\n"
	s += renderField("Type: ", row.Type) + "\n"
	s += renderField("Amount: ", row.AmountInput+" "+currency) + "\n"
	s += renderField("Account: ", account) + "\n"
	if category != "" {
		s += renderField("Category: ", category) + "\n"
	}
	if row.Note != "" {
		s += renderField("Note: ", row.Note) + "\n"
	}
	if warning != "" {
		s += "\n" + warnStyle.Render(warning) + "\n"
	}
	return s
}

// runRowEdit lets the user revise amount, date, and note for one row, clearing
// the parse error if the edited values are now valid.
func runRowEdit(row importer.ParsedRow) (importer.ParsedRow, error) {
	for {
		items := []selectionItem[string]{
			{label: "Amount (" + row.AmountInput + ")", value: "amount"},
			{label: "Date (" + row.Date + ")", value: "date"},
			{label: "Note (" + row.Note + ")", value: "note"},
			{label: "Done", value: "done"},
		}
		choice, chose, err := runSelection(HeaderStyle.Render("Edit row"), "", items)
		if err != nil {
			return row, err
		}
		if !chose || choice == "done" {
			return revalidateRow(row), nil
		}

		switch choice {
		case "amount":
			value, ok, err := runTextPrompt("Edit amount", "Amount: ", row.AmountInput, validateAmount)
			if err != nil {
				return row, err
			}
			if ok {
				row.AmountInput = value
			}
		case "date":
			value, ok, err := runTextPrompt("Edit date", "Date (YYYY-MM-DD): ", row.Date, validateDate)
			if err != nil {
				return row, err
			}
			if ok {
				row.Date = value
			}
		case "note":
			value, ok, err := runTextPrompt("Edit note", "Note: ", row.Note, nil)
			if err != nil {
				return row, err
			}
			if ok {
				row.Note = value
			}
		}
	}
}

// revalidateRow recomputes ParseErr after edits so a row that was flagged on
// import can clear its error once amount and date are valid.
func revalidateRow(row importer.ParsedRow) importer.ParsedRow {
	row.ParseErr = nil
	if err := validateAmount(row.AmountInput); err != nil {
		row.ParseErr = err
		return row
	}
	if err := validateDate(row.Date); err != nil {
		row.ParseErr = err
	}
	return row
}

func validateAmount(s string) error {
	amountMinor, err := money.Parse(s)
	if err != nil {
		return err
	}
	if amountMinor <= 0 {
		return fmt.Errorf("amount must be greater than zero")
	}
	return nil
}

func validateDate(s string) error {
	if _, err := time.Parse("2006-01-02", strings.TrimSpace(s)); err != nil {
		return fmt.Errorf("invalid date format: %s", s)
	}
	return nil
}

// promptAccount asks the user to pick the account that imported rows post to,
// returning that account's name and currency. The import currency is derived from
// the chosen account rather than asked separately: an existing account has a fixed
// currency, and posting a differing currency would make every save fail the
// account-currency check in the service layer. Currency is only entered when the
// user creates a new account.
func promptAccount() (string, string, bool, error) {
	accounts, err := Svc.LoadAccounts()
	if err != nil {
		return "", "", false, err
	}

	items := make([]selectionItem[string], 0, len(accounts)+1)
	for _, account := range accounts {
		items = append(items, selectionItem[string]{
			label: account.Name + " (" + account.Currency + ")",
			value: account.Name,
		})
	}
	items = append(items, selectionItem[string]{label: "[New account]", value: newAccountSentinel})

	choice, chose, err := runSelection(HeaderStyle.Render("Import setup"), "Account for imported rows:", items)
	if err != nil || !chose {
		return "", "", chose, err
	}
	if choice != newAccountSentinel {
		for _, account := range accounts {
			if account.Name == choice {
				return account.Name, account.Currency, true, nil
			}
		}
	}

	return promptNewAccount(accounts)
}

func promptNewAccount(existing []model.Account) (string, string, bool, error) {
	nameValidate := func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("account name is required")
		}
		return nil
	}
	name, ok, err := runTextPrompt(HeaderStyle.Render("New account"), "Account name: ", "", nameValidate)
	if err != nil || !ok {
		return "", "", ok, err
	}

	for _, account := range existing {
		if strings.EqualFold(account.Name, name) {
			return account.Name, account.Currency, true, nil
		}
	}

	currencyValidate := func(s string) error {
		if money.NormalizeCurrency(s) == "" {
			return fmt.Errorf("currency is required")
		}
		return nil
	}
	currencyInput, ok, err := runTextPrompt(HeaderStyle.Render("New account"), "Currency: ", "CAD", currencyValidate)
	if err != nil || !ok {
		return "", "", ok, err
	}
	currency := money.NormalizeCurrency(currencyInput)

	if _, err := Svc.AddAccount(name, currency, "0"); err != nil {
		return "", "", false, err
	}
	return name, currency, true, nil
}

// runCategoryPicker reproduces the add flow's select-or-create category UX from
// the shared prompt primitives: pick an existing category or enter a new one,
// confirming creation of an unknown name.
func runCategoryPicker(categories []string) (string, bool, error) {
	items := make([]selectionItem[string], 0, len(categories)+1)
	for _, c := range categories {
		items = append(items, selectionItem[string]{label: c, value: c})
	}
	const newCategorySentinel = "\x00new-category"
	items = append(items, selectionItem[string]{label: "[New category]", value: newCategorySentinel})

	choice, chose, err := runSelection(HeaderStyle.Render("Category"), "Select category:", items)
	if err != nil || !chose {
		return "", chose, err
	}
	if choice != newCategorySentinel {
		return choice, true, nil
	}

	for {
		validate := func(s string) error {
			if strings.TrimSpace(s) == "" {
				return fmt.Errorf("category is required")
			}
			return nil
		}
		draft, ok, err := runTextPrompt(HeaderStyle.Render("New category"), "Category: ", "", validate)
		if err != nil || !ok {
			return "", ok, err
		}
		if containsFold(categories, draft) {
			return draft, true, nil
		}
		create, err := RunConfirmPrompt(warnStyle.Render("Category '" + draft + "' is new. Create it?"))
		if err != nil {
			return "", false, err
		}
		if create {
			return draft, true, nil
		}
	}
}
