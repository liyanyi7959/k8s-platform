package application

import (
	"fmt"
	"strings"
)

// ActionPayload extracts the optional action payload from a persisted change
// document. Missing or incompatible values intentionally behave as an empty
// payload so callers can safely inspect optional action attributes.
func ActionPayload(change map[string]any) map[string]any {
	payload, ok := change["payload"].(map[string]any)
	if !ok || payload == nil {
		return map[string]any{}
	}
	return payload
}

// ActionExecutionStatusLabel provides the user-facing label for a completed
// action execution. The storage status remains the machine-readable value.
func ActionExecutionStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "succeeded":
		return "成功"
	case "failed":
		return "失败"
	default:
		return "完成"
	}
}

// IntegerInputValue accepts the numeric forms produced by JSON decoding and
// normalizes them for tool and action inputs. Unsupported values are absent.
func IntegerInputValue(value any) (int, bool) {
	switch number := value.(type) {
	case int:
		return number, true
	case int8:
		return int(number), true
	case int16:
		return int(number), true
	case int32:
		return int(number), true
	case int64:
		return int(number), true
	case uint:
		return int(number), true
	case uint8:
		return int(number), true
	case uint16:
		return int(number), true
	case uint32:
		return int(number), true
	case uint64:
		return int(number), true
	case float32:
		return int(number), true
	case float64:
		return int(number), true
	default:
		return 0, false
	}
}

// NestedIntegerInputValue resolves an optional integer at a JSON object path.
// A missing path or a non-object intermediate node returns zero.
func NestedIntegerInputValue(obj map[string]any, keys ...string) int {
	if len(keys) == 0 {
		return 0
	}
	current := obj
	for _, key := range keys[:len(keys)-1] {
		next, _ := current[key].(map[string]any)
		if next == nil {
			return 0
		}
		current = next
	}
	value, ok := IntegerInputValue(current[keys[len(keys)-1]])
	if !ok {
		return 0
	}
	return value
}

// StringInputValue turns a JSON input scalar into normalized display text.
func StringInputValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []byte:
		return strings.TrimSpace(string(typed))
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}
