package discover

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"unicode"

	"github.com/esper-io/esper-cli/internal/cmd/generated"
	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
	"github.com/spf13/cobra"
)

type result struct {
	Command     string   `json:"command"`
	Summary     string   `json:"summary"`
	OperationID string   `json:"operation_id"`
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	Docs        []string `json:"docs,omitempty"`
	score       int
}

func NewCommand(options *esperruntime.GlobalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "discover <query-or-docs-url>",
		Short: "Find Esper operations and commands",
		Long:  "Find generated Esper commands from names, OpenAPI operation IDs, paths, request fields, or public documentation URLs.",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			matches := search(generated.Operations(), args[0])
			if len(matches) == 0 {
				return esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("no commands found for %q", args[0]))
			}
			if options.JSON {
				return json.NewEncoder(command.OutOrStdout()).Encode(matches)
			}
			for _, match := range matches {
				if _, err := fmt.Fprintf(command.OutOrStdout(), "espercli %s\n  %s %s\n  %s\n", match.Command, match.Method, match.Path, match.Summary); err != nil {
					return err
				}
				for _, docsURL := range match.Docs {
					if _, err := fmt.Fprintf(command.OutOrStdout(), "  Docs: %s\n", docsURL); err != nil {
						return err
					}
				}
				if _, err := fmt.Fprintln(command.OutOrStdout(), "  Run with --help for arguments and request-body details."); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func search(operations []generated.Operation, query string) []result {
	slug := docsSlug(query)
	words := searchWords(query)
	byCommand := map[string]result{}
	for _, operation := range operations {
		if operation.AliasOf != "" {
			continue
		}
		command := strings.Join(operation.Command, " ")
		score := matchScore(operation, command, slug, words)
		if score == 0 {
			continue
		}
		candidate := result{
			Command:     command,
			Summary:     operation.Summary,
			OperationID: operation.OperationID,
			Method:      operation.Method,
			Path:        operation.Path,
			Docs:        docsURLs(operation.DocsSlugs),
			score:       score,
		}
		if existing, ok := byCommand[command]; !ok || candidate.score > existing.score {
			byCommand[command] = candidate
		}
	}
	results := make([]result, 0, len(byCommand))
	for _, match := range byCommand {
		results = append(results, match)
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].score != results[j].score {
			return results[i].score > results[j].score
		}
		return results[i].Command < results[j].Command
	})
	for index := range results {
		results[index].score = 0
	}
	return results
}

func matchScore(operation generated.Operation, command, slug string, words []string) int {
	for _, candidate := range operation.DocsSlugs {
		if slug != "" && strings.EqualFold(candidate, slug) {
			return 100
		}
	}
	primary := []string{command, operation.OperationID, operation.Path, operation.Summary, operation.Description, strings.Join(operation.Tags, " "), strings.Join(operation.DocsSlugs, " ")}
	query := make([]string, 0)
	body := make([]string, 0)
	for _, parameter := range operation.Parameters {
		metadata := []string{parameter.Name, parameter.Description, strings.Join(parameter.Enum, " ")}
		if parameter.In == "query" {
			query = append(query, metadata...)
		}
	}
	if operation.Body != nil {
		for _, field := range operation.Body.Fields {
			body = append(body, field.Path, field.Description, strings.Join(field.Enum, " "))
		}
	}
	if len(words) == 0 {
		return 0
	}
	if strings.EqualFold(command, strings.TrimSpace(strings.Join(words, " "))) || strings.EqualFold(operation.OperationID, strings.TrimSpace(strings.Join(words, " "))) {
		return 80
	}
	if containsAll(strings.Join(append(primary, discoverAliases(operation)...), " "), words) {
		return 70
	}
	if containsAll(strings.Join(append(primary, query...), " "), words) {
		return 60
	}
	if containsAll(strings.Join(append(append(primary, query...), body...), " "), words) {
		return 40
	}
	return 0
}

func containsAll(value string, words []string) bool {
	value = strings.ToLower(value)
	for _, word := range words {
		if !strings.Contains(value, word) {
			return false
		}
	}
	return true
}

func discoverAliases(operation generated.Operation) []string {
	if strings.Join(operation.Command, " ") == "device list" {
		return []string{"active inactive disabled state 1 20 60"}
	}
	return nil
}

func docsSlug(value string) string {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return ""
	}
	const prefix = "/openapi/"
	if !strings.HasPrefix(parsed.Path, prefix) {
		return ""
	}
	return strings.Trim(strings.TrimPrefix(parsed.Path, prefix), "/")
}

func searchWords(value string) []string {
	return strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

func docsURLs(slugs []string) []string {
	result := make([]string, len(slugs))
	for index, slug := range slugs {
		result[index] = "https://api.esper.io/openapi/" + slug
	}
	return result
}
