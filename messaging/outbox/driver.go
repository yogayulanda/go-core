package outbox

import "strings"

func normalizeDriver(driver string) string {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "postgres":
		return "postgres"
	case "sqlserver":
		return "sqlserver"
	case "mysql":
		return "mysql"
	default:
		return "mysql"
	}
}

func keyColumnByDriver(driver string) string {
	switch normalizeDriver(driver) {
	case "postgres":
		return `"key"`
	case "sqlserver":
		return "[key]"
	default:
		return "`key`"
	}
}
