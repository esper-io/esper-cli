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
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	table := tabwriter.NewWriter(writer, 0, 4, 2, ' ', 0)
	for _, key := range keys {
		if _, err := fmt.Fprintf(table, "%s\t%s\n", key, printable(values[key])); err != nil {
			return err
		}
	}
	return table.Flush()
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
			_, err := fmt.Fprintln(writer, printable(row))
			return err
		}
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
	return table.Flush()
}

func printable(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case float64, bool:
		return fmt.Sprint(typed)
	default:
		var buffer bytes.Buffer
		encoder := json.NewEncoder(&buffer)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(typed); err != nil {
			return fmt.Sprint(typed)
		}
		return strings.TrimSpace(buffer.String())
	}
}
