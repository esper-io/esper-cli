package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
)

func WriteJSON(writer io.Writer, data []byte) error {
	if len(bytes.TrimSpace(data)) == 0 {
		_, err := fmt.Fprintln(writer, "null")
		return err
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("decode JSON output: %w", err)
	}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("write JSON output: %w", err)
	}
	return nil
}

func UnwrapResponseEnvelope(data []byte, envelope string) ([]byte, error) {
	if envelope != "apps-envelope" || len(bytes.TrimSpace(data)) == 0 {
		return data, nil
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		return nil, fmt.Errorf("decode %s response: %w", envelope, err)
	}
	content, ok := object["content"]
	if !ok {
		return data, nil
	}
	return content, nil
}

func WriteHuman(writer io.Writer, data []byte) error {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("decode human output: %w", err)
	}
	switch typed := value.(type) {
	case []any:
		return writeTable(writer, typed)
	case map[string]any:
		return writeKeyValues(writer, typed)
	default:
		_, err := fmt.Fprintln(writer, typed)
		return err
	}
}

func writeKeyValues(writer io.Writer, values map[string]any) error {
	values = flattenObject(values)
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	table := tabwriter.NewWriter(writer, 0, 4, 2, ' ', 0)
	var sections []string
	for _, key := range keys {
		if rows, ok := values[key].([]any); ok && tableSection(rows) {
			sections = append(sections, key)
			continue
		}
		if _, err := fmt.Fprintf(table, "%s\t%s\n", key, printable(values[key])); err != nil {
			return err
		}
	}
	if err := table.Flush(); err != nil {
		return err
	}
	for _, key := range sections {
		if _, err := fmt.Fprintf(writer, "\n%s:\n", key); err != nil {
			return err
		}
		if err := writeTable(writer, values[key].([]any)); err != nil {
			return err
		}
	}
	return nil
}

func tableSection(rows []any) bool {
	if len(rows) == 0 {
		return true
	}
	for _, row := range rows {
		if _, ok := row.(map[string]any); ok {
			return true
		}
	}
	return false
}

func flattenObject(object map[string]any) map[string]any {
	values := make(map[string]any)
	for key, value := range object {
		if nested, ok := value.(map[string]any); ok && len(nested) != 0 {
			for name, child := range flattenObject(nested) {
				values[key+"."+name] = child
			}
		} else {
			values[key] = value
		}
	}
	return values
}

func writeTable(writer io.Writer, rows []any) error {
	if len(rows) == 0 {
		_, err := fmt.Fprintln(writer, "No results.")
		return err
	}
	objects := make([]map[string]any, 0, len(rows))
	columns := map[string]struct{}{}
	for _, row := range rows {
		object, ok := row.(map[string]any)
		if !ok {
			for _, item := range rows {
				if _, err := fmt.Fprintln(writer, printable(item)); err != nil {
					return err
				}
			}
			return nil
		}
		object = flattenObject(object)
		objects = append(objects, object)
		for key := range object {
			columns[key] = struct{}{}
		}
	}
	keys := make([]string, 0, len(columns))
	for key := range columns {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	totalColumns := len(keys)
	if len(keys) > 8 {
		selected := make([]string, 0, 8)
		seen := map[string]bool{}
		for _, key := range append([]string{"id", "name", "status", "state", "platform", "operation_type", "hardware_info.model", "last_seen", "alias", "type", "version", "created_at"}, keys...) {
			if _, exists := columns[key]; exists && !seen[key] {
				selected = append(selected, key)
				seen[key] = true
				if len(selected) == 8 {
					break
				}
			}
		}
		keys = selected
	}
	table := tabwriter.NewWriter(writer, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(table, strings.Join(keys, "\t")); err != nil {
		return err
	}
	for _, object := range objects {
		values := make([]string, len(keys))
		for index, key := range keys {
			values[index] = printable(object[key])
		}
		if _, err := fmt.Fprintln(table, strings.Join(values, "\t")); err != nil {
			return err
		}
	}
	if err := table.Flush(); err != nil {
		return err
	}
	if totalColumns > len(keys) {
		_, err := fmt.Fprintf(writer, "Showing %d of %d fields. Use --json for the full response.\n", len(keys), totalColumns)
		return err
	}
	return nil
}

func printable(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(typed)
	case float64, bool:
		return fmt.Sprint(typed)
	case []any:
		values := make([]string, len(typed))
		for index, value := range typed {
			values[index] = printable(value)
		}
		return strings.Join(values, ", ")
	case map[string]any:
		values := flattenObject(typed)
		keys := make([]string, 0, len(values))
		for key := range values {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			parts = append(parts, key+"="+printable(values[key]))
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprint(typed)
	}
}
