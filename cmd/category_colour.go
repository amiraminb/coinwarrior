package cmd

import (
	"hash/fnv"
	"slices"
	"strings"

	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/charmbracelet/lipgloss"
)

// Five hues validated all-pairs on light and dark surfaces (CVD ΔE 9.2); a
// sixth drops to 5.6 and no seventh clears the normal-vision floor.
var categoryColours = []lipgloss.Style{
	lipgloss.NewStyle().Foreground(lipgloss.Color("68")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("162")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("129")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("28")),
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
