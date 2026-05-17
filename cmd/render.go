package cmd

func renderField(label, value string) string {
	return label + valueStyle.Render(value)
}

func renderActiveField(label, value string) string {
	rendered := ""
	if value != "" {
		rendered = valueStyle.Render(value)
	}
	return label + rendered + cursorStyle.Render(" ")
}

func renderError(message string) string {
	if message == "" {
		return ""
	}
	return warnStyle.Render(message) + "\n"
}
