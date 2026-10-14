package app

import (
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/yogayulanda/go-core/config"
	"github.com/yogayulanda/go-core/logger"
)

var durationType = reflect.TypeOf(time.Duration(0))

func effectiveConfigFields(cfg *config.Config) []logger.Field {
	if cfg == nil {
		return nil
	}
	var fields []logger.Field
	collectConfigFields(reflect.ValueOf(*cfg), "", &fields)
	sort.Slice(fields, func(i, j int) bool { return fields[i].Key < fields[j].Key })
	return fields
}

func collectConfigFields(value reflect.Value, prefix string, fields *[]logger.Field) {
	if value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			*fields = append(*fields, logger.Field{Key: strings.TrimSuffix(prefix, "."), Value: nil})
			return
		}
		collectConfigFields(value.Elem(), prefix, fields)
		return
	}
	if value.Type() == durationType {
		*fields = append(*fields, logger.Field{Key: strings.TrimSuffix(prefix, "."), Value: time.Duration(value.Int()).String()})
		return
	}
	switch value.Kind() {
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			fieldType := value.Type().Field(i)
			if fieldType.PkgPath != "" {
				continue
			}
			collectConfigFields(value.Field(i), prefix+snakeCase(fieldType.Name)+".", fields)
		}
	case reflect.Map:
		if value.IsNil() {
			*fields = append(*fields, logger.Field{Key: strings.TrimSuffix(prefix, "."), Value: nil})
			return
		}
		keys := value.MapKeys()
		sort.Slice(keys, func(i, j int) bool { return keys[i].String() < keys[j].String() })
		for _, key := range keys {
			collectConfigFields(value.MapIndex(key), prefix+snakeCase(key.String())+".", fields)
		}
	case reflect.Slice, reflect.Array:
		items := make([]interface{}, value.Len())
		for i := 0; i < value.Len(); i++ {
			items[i] = value.Index(i).Interface()
		}
		appendConfigValue(prefix, items, fields)
	default:
		if value.CanInterface() {
			appendConfigValue(prefix, value.Interface(), fields)
		}
	}
}

func appendConfigValue(prefix string, value interface{}, fields *[]logger.Field) {
	key := strings.TrimSuffix(prefix, ".")
	if isSecretConfigField(key) {
		value = "<REDACTED>"
	}
	*fields = append(*fields, logger.Field{Key: key, Value: value})
}

func isSecretConfigField(key string) bool {
	key = strings.ToLower(strings.ReplaceAll(key, "_", ""))
	for _, marker := range []string{"password", "masterkey", "privatekey", "keyfile", "secret", "dsn", "jksfile"} {
		if strings.Contains(key, marker) {
			return true
		}
	}
	return false
}

func snakeCase(value string) string {
	var out strings.Builder
	runes := []rune(value)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 && (unicode.IsLower(runes[i-1]) || unicode.IsDigit(runes[i-1]) || (i+1 < len(runes) && unicode.IsLower(runes[i+1]))) {
				out.WriteByte('_')
			}
			out.WriteRune(unicode.ToLower(r))
		} else {
			out.WriteRune(r)
		}
	}
	return out.String()
}
