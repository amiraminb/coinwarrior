package cmd

import (
	"hash/fnv"
	"slices"
	"strings"

	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/charmbracelet/lipgloss"
)

var categoryColours = []lipgloss.Style{
	lipgloss.NewStyle().Foreground(lipgloss.Color("146")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("101")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("179")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("68")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("174")),
}

var transferCategoryStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

// Positional, not hashed: hashing clustered five of ten categories onto one hue.
func categoryStyle(category string) lipgloss.Style {
	clean := strings.TrimSpace(category)
	if clean == "" || strings.EqualFold(clean, model.TransferCategory) {
		return transferCategoryStyle
	}

	saved, err := svc.LoadCategories()
	if err == nil {
		if i := slices.IndexFunc(saved, func(c string) bool { return strings.EqualFold(c, clean) }); i >= 0 {
			return categoryColours[i%len(categoryColours)]
		}
	}

	h := fnv.New32a()
	h.Write([]byte(strings.ToLower(clean)))
	return categoryColours[h.Sum32()%uint32(len(categoryColours))]
}

func colourCategory(category string) string {
	return categoryStyle(category).Render(category)
}
