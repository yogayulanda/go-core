package config

import "strings"

func NormalizeDBAlias(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func DatabaseEnvPrefix(name string) string {
	return "DB_" + strings.ToUpper(strings.TrimSpace(name)) + "_"
}

func hasDatabaseAlias(databases map[string]DBConfig, name string) bool {
	name = NormalizeDBAlias(name)
	for alias := range databases {
		if NormalizeDBAlias(alias) == name {
			return true
		}
	}
	return false
}
