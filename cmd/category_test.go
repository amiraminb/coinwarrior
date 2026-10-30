package cmd

import (
	"bytes"
	"testing"

	"github.com/amiraminb/coinwarrior/internal/repository"
	"github.com/amiraminb/coinwarrior/internal/service"
)

func TestCategoryCommands(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	r := repository.NewFileRepository()
	if err := r.SaveCategories([]string{"Groceries"}); err != nil {
		t.Fatal(err)
	}
	previous := svc
	svc = service.New(r)
	t.Cleanup(func() { svc = previous })
	run := func(args ...string) (string, error) {
		cmd := newCategoryCommand()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs(args)
		err := cmd.Execute()
		return out.String(), err
	}
	for _, args := range [][]string{
		{"add", " Food Delivery "},
		{"edit", "food delivery", "Meal Delivery"},
	} {
		if _, err := run(args...); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{nil, {"list"}} {
		out, err := run(args...)
		if err != nil || out != "Groceries\nMeal Delivery\n" {
			t.Fatalf("list = %q, error = %v", out, err)
		}
	}
	for _, args := range [][]string{{"add", "groceries"}, {"add", " "}, {"rename", "Missing", "New"}, {"delete", "Missing"}} {
		if _, err := run(args...); err == nil {
			t.Errorf("%v should fail", args)
		}
	}
	if _, err := run("delete", "meal delivery"); err != nil {
		t.Fatal(err)
	}
	if out, err := run("list"); err != nil || out != "Groceries\n" {
		t.Fatalf("list after deletion = %q, error = %v", out, err)
	}
}
