package outbox

import "strings"

func normalizeDriver(driver string) string {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "sqlserver":
		return "sqlserver"
	case "mysql":
		return "mysql"
	default:
		return ""
	}
}

func keyColumnByDriver(driver string) string {
	switch normalizeDriver(driver) {
	case "sqlserver":
		return "[key]"
	case "mysql":
		return "`key`"
	default:
		return ""
	}
}
