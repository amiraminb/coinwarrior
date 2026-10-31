package service

import (
	"reflect"
	"slices"
	"testing"
	"time"
)

func TestPendingRecurringCountMatchesGenerationWithoutWriting(t *testing.T) {
	now := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name, start, end, last string
		want                   int
	}{
		{"new month before due day", "2026-10-01", "", "2026-10", 1},
		{"already generated", "2026-10-01", "", "2026-11", 0},
		{"future rule", "2026-12-01", "", "", 0},
		{"ended rule caught up", "2026-10-01", "2026-10-31", "2026-10", 0},
		{"ended rule with backlog", "2026-10-01", "2026-10-31", "", 1},
		{"multiple missing months", "2026-09-01", "", "2026-09", 2},
		{"starts after scheduled day", "2026-11-29", "", "", 0},
		{"ends before scheduled day", "2026-11-01", "2026-11-10", "", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rule := expenseRule(28, tc.start, tc.end)
			rule.LastGeneratedMonth = tc.last
			r := ruleRepo(rule)
			before := *r
			before.rules = slices.Clone(r.rules)
			before.accounts = slices.Clone(r.accounts)
			s := New(r)
			count, err := s.PendingRecurringTransactionCount(now)
			if err != nil || count != tc.want {
				t.Fatalf("pending = %d, %v; want %d", count, err, tc.want)
			}
			if !reflect.DeepEqual(*r, before) {
				t.Fatal("checking pending transactions changed the ledger, accounts, or rules")
			}
			generated, err := s.GenerateDueTransactions(now)
			if err != nil || len(generated.Generated) != count {
				t.Fatalf("generation disagrees with pending count: %+v, %v", generated, err)
			}
			if remaining, err := s.PendingRecurringTransactionCount(now); err != nil || remaining != 0 {
				t.Fatalf("pending after generation = %d, %v", remaining, err)
			}
		})
	}
}

func TestPendingRecurringCountEmptyAndInvalidRules(t *testing.T) {
	r := &fakeRepo{}
	s := New(r)
	if count, err := s.PendingRecurringTransactionCount(generationNow); err != nil || count != 0 {
		t.Fatalf("empty rules = %d, %v", count, err)
	}
	r.rules = append(r.rules, expenseRule(1, "invalid", ""))
	if _, err := s.PendingRecurringTransactionCount(generationNow); err == nil {
		t.Fatal("invalid schedule should report an error")
	}
}
