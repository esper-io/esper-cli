package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/esper-io/esper-cli/internal/cmd"
	"github.com/esper-io/esper-cli/internal/cmd/generated"
	"github.com/spf13/cobra"
)

var methods = map[string]bool{"get": true, "post": true, "put": true, "patch": true, "delete": true}
var coreGenerationRank = map[string]int{"legacy": 0, "v0": 1, "v1": 2, "v2": 3, "v3": 4}
var sideFamilyPrefixes = map[string]string{"authn2": "authn", "authz2": "authz", "foundry": "foundry", "geofence": "geofence", "pipelines-v0": "pipeline"}

type spec struct {
	Info struct {
		Generation string `json:"x-esper-generation"`
	} `json:"info"`
	Paths      map[string]map[string]operation `json:"paths"`
	Components struct {
		Parameters map[string]parameter `json:"parameters"`
	} `json:"components"`
}
type operation struct {
	Parameters   []parameter `json:"parameters"`
	Destructive  *bool       `json:"x-esper-destructive"`
	Pagination   string      `json:"x-esper-pagination"`
	Verb         string      `json:"x-esper-verb"`
	Noun         string      `json:"x-esper-noun"`
	ScopeParent  string      `json:"x-esper-scope-parent"`
	Summary      string      `json:"summary"`
	OperationID  string      `json:"operationId"`
	AliasOf      string      `json:"x-esper-alias-of"`
	RequireOneOf []string    `json:"x-esper-require-one-of"`
	Body         struct {
		Required bool `json:"required"`
		Content  map[string]struct {
			Schema struct {
				Properties map[string]json.RawMessage `json:"properties"`
			} `json:"schema"`
		} `json:"content"`
	} `json:"requestBody"`
}
type parameter struct {
	Ref      string `json:"$ref"`
	Name     string `json:"name"`
	In       string `json:"in"`
	Required bool   `json:"required"`
}

func main() {
	specDir := flag.String("spec-dir", "spec/openapi", "directory containing canonical OpenAPI files")
	flag.Parse()
	issues := check(*specDir)
	if len(issues) == 0 {
		fmt.Println("contract check passed")
		return
	}
	sort.Strings(issues)
	for _, issue := range issues {
		fmt.Fprintln(os.Stderr, "error:", issue)
	}
	os.Exit(1)
}

func check(specDir string) []string {
	byOperation := map[string]generated.Operation{}
	for _, operation := range generated.Operations() {
		byOperation[key(operation.Generation, operation.Method, operation.Path)] = operation
	}
	entries, err := os.ReadDir(specDir)
	if err != nil {
		return []string{fmt.Sprintf("read spec directory: %v", err)}
	}
	root := cmd.NewRootCommand()
	issues := []string{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(specDir, entry.Name()))
		if err != nil {
			return []string{fmt.Sprintf("read %s: %v", entry.Name(), err)}
		}
		var document spec
		if err := json.Unmarshal(data, &document); err != nil {
			return []string{fmt.Sprintf("decode %s: %v", entry.Name(), err)}
		}
		for apiPath, item := range document.Paths {
			for method, operation := range item {
				if !methods[method] {
					continue
				}
				location := fmt.Sprintf("%s %s %s", strings.ToUpper(method), apiPath, document.Info.Generation)
				if operation.Destructive == nil || operation.Pagination == "" || operation.Verb == "" || operation.Noun == "" {
					issues = append(issues, location+": missing required x-esper annotation")
					continue
				}
				if strings.TrimSpace(operation.Summary) == "" {
					issues = append(issues, location+": missing operation summary")
				}
				if hasVersionSuffix(operation.Noun) || hasPluralComponent(operation.Noun) {
					issues = append(issues, location+": overlay noun violates CLI grammar: "+operation.Noun)
				}
				generatedOperation, ok := byOperation[key(document.Info.Generation, strings.ToUpper(method), apiPath)]
				if !ok {
					issues = append(issues, location+": no generated operation")
					continue
				}
				expectedNoun := operation.Noun
				if prefix, side := sideFamilyPrefixes[document.Info.Generation]; side && generatedOperation.Noun == prefix+"-"+operation.Noun {
					expectedNoun = generatedOperation.Noun
				}
				if generatedOperation.Noun != expectedNoun || generatedOperation.Verb != operation.Verb || generatedOperation.Destructive != *operation.Destructive || generatedOperation.Summary != operation.Summary || generatedOperation.OperationID != operation.OperationID || generatedOperation.AliasOf != operation.AliasOf || !sameStrings(generatedOperation.RequireOneOf, operation.RequireOneOf) {
					issues = append(issues, location+": generated metadata differs from overlay")
				}
				if operation.Body.Required && supportedBody(operation.Body.Content) && generatedOperation.Body == nil {
					issues = append(issues, location+": required request body missing generated metadata")
				} else if generatedOperation.Body != nil && generatedOperation.Body.Required != operation.Body.Required {
					issues = append(issues, location+": generated request body requiredness differs from spec")
				}
				command, _, err := root.Find(generatedOperation.Command)
				if err != nil || command == root {
					issues = append(issues, location+": generated command is unreachable")
					continue
				}
				if strings.TrimSpace(command.Short) == "" {
					issues = append(issues, location+": generated command has no description")
				}
				for index := 1; index < len(generatedOperation.Command)-1; index++ {
					groupPath := generatedOperation.Command[:index]
					group, _, err := root.Find(groupPath)
					expectedSummary := "Commands in the " + generatedOperation.Command[index-1] + " group"
					if err != nil || group == root || group.Short != expectedSummary {
						issues = append(issues, location+": generated group has no explicit description")
					}
				}
				if operation.AliasOf != "" {
					continue
				}
				for _, parameter := range operation.Parameters {
					parameter = resolveParameter(parameter, document.Components.Parameters)
					if parameter.In == "path" && parameter.Name != operation.ScopeParent && operation.ScopeParent == "" {
						continue
					}
					if parameter.In == "path" && operation.ScopeParent != "" {
						continue
					}
					if parameter.In == "header" || parameter.In == "query" {
						if command.Flags().Lookup(kebab(parameter.Name)) == nil {
							issues = append(issues, location+": missing --"+kebab(parameter.Name))
						}
					}
				}
				if *operation.Destructive && root.PersistentFlags().Lookup("yes") == nil {
					issues = append(issues, location+": destructive command lacks --yes")
				}
			}
		}
	}
	issues = append(issues, checkCommandGrammar(root)...)
	issues = append(issues, checkDescriptions(root)...)
	return issues
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func checkDescriptions(root *cobra.Command) []string {
	issues := []string{}
	var walk func(*cobra.Command)
	walk = func(command *cobra.Command) {
		for _, child := range command.Commands() {
			if child.Name() == "help" || child.Name() == "completion" {
				continue
			}
			if strings.TrimSpace(child.Short) == "" {
				issues = append(issues, "generated command has no description: "+child.CommandPath())
			}
			walk(child)
		}
	}
	walk(root)
	return issues
}

func checkCommandGrammar(root interface {
	Find([]string) (*cobra.Command, []string, error)
}) []string {
	operations := generated.Operations()
	issues := []string{}
	byNounVerb := map[string][]generated.Operation{}
	byCommand := map[string][]generated.Operation{}
	for _, operation := range operations {
		if operation.AliasOf != "" {
			continue
		}
		byNounVerb[operation.Noun+"\x00"+operation.Verb] = append(byNounVerb[operation.Noun+"\x00"+operation.Verb], operation)
		byCommand[strings.Join(operation.Command, "\x00")] = append(byCommand[strings.Join(operation.Command, "\x00")], operation)
	}
	for _, group := range byNounVerb {
		maxRank := -1
		for _, operation := range group {
			if rank, core := coreGenerationRank[operation.Generation]; core && rank > maxRank {
				maxRank = rank
			}
		}
		for _, operation := range group {
			rank, core := coreGenerationRank[operation.Generation]
			var expected []string
			if core && rank < maxRank {
				expected = []string{"api", operation.Generation, operation.Noun, operation.Verb}
			} else {
				expected = []string{operation.Noun, operation.Verb}
			}
			if strings.Join(operation.Command, "\x00") != strings.Join(expected, "\x00") {
				issues = append(issues, fmt.Sprintf("%s %s %s: violates generation collision namespace", operation.Method, operation.Path, operation.Generation))
			}
		}
	}
	for commandPath, group := range byCommand {
		scopes := map[string]bool{}
		scopeNames := map[string]bool{}
		unscoped := 0
		for _, operation := range group {
			names := operationScopeNames(operation)
			for _, name := range names {
				scopeNames[name] = true
			}
			scope := strings.Join(names, "\x00")
			if scope == "" {
				unscoped++
				continue
			}
			if scopes[scope] {
				issues = append(issues, commandPath+": duplicate scope combination "+formatScopeNames(operationScopeNames(operation)))
				continue
			}
			scopes[scope] = true
		}
		if unscoped > 1 {
			issues = append(issues, commandPath+": multiple unscoped operations")
		}
		command, _, err := root.Find(strings.Split(commandPath, "\x00"))
		if err != nil || command == nil {
			continue
		}
		for scope := range scopeNames {
			if command.Flags().Lookup(kebab(scope)) == nil {
				issues = append(issues, commandPath+": missing scope flag --"+kebab(scope))
			}
		}
	}
	for _, operation := range operations {
		if operation.AliasOf != "" {
			if operation.OperationID == "" {
				issues = append(issues, operation.Method+" "+operation.Path+": alias lacks operation ID")
			}
			continue
		}
		if operation.Generation != "pipelines-v0" {
			continue
		}
		if noun := relationshipNoun(operation.Path); noun != "" && operation.Noun != noun {
			issues = append(issues, operation.Method+" "+operation.Path+": synthesized relationship noun "+operation.Noun+", want "+noun)
		}
	}
	return issues
}

func operationScopeNames(operation generated.Operation) []string {
	var names []string
	seen := map[string]bool{}
	for _, parameter := range operation.Parameters {
		if parameter.In == "path" && parameter.Scope {
			name := parameter.ScopeName
			if name == "" {
				name = operation.ScopeParent
			}
			if name != "" && !seen[name] {
				names = append(names, name)
				seen[name] = true
			}
		}
	}
	sort.Strings(names)
	return names
}

func formatScopeNames(names []string) string {
	flags := make([]string, 0, len(names))
	for _, name := range names {
		flags = append(flags, "--"+kebab(name))
	}
	return strings.Join(flags, " + ")
}

func supportedBody(content map[string]struct {
	Schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
	} `json:"schema"`
}) bool {
	_, jsonBody := content["application/json"]
	_, multipartBody := content["multipart/form-data"]
	return jsonBody || multipartBody
}

func key(generation, method, apiPath string) string {
	return generation + "\x00" + method + "\x00" + apiPath
}
func resolveParameter(value parameter, parameters map[string]parameter) parameter {
	if !strings.HasPrefix(value.Ref, "#/components/parameters/") {
		return value
	}
	name := strings.TrimPrefix(value.Ref, "#/components/parameters/")
	if resolved, ok := parameters[name]; ok {
		return resolveParameter(resolved, parameters)
	}
	return value
}
func kebab(value string) string {
	var result []rune
	for index, char := range value {
		if char == '_' {
			result = append(result, '-')
			continue
		}
		if index > 0 && char >= 'A' && char <= 'Z' {
			result = append(result, '-')
		}
		result = append(result, char)
	}
	return strings.ToLower(string(result))
}

func hasVersionSuffix(noun string) bool {
	parts := strings.Split(noun, "-")
	for _, part := range parts {
		if len(part) > 1 && part[0] == 'v' && allDigits(part[1:]) {
			return true
		}
	}
	return false
}
func allDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}
func hasPluralComponent(noun string) bool {
	for _, part := range strings.Split(noun, "-") {
		if strings.HasSuffix(part, "s") && !strings.HasSuffix(part, "ss") && part != "status" && part != "apns" {
			return true
		}
	}
	return false
}

func relationshipNoun(apiPath string) string {
	segments := []string{}
	for _, segment := range strings.Split(apiPath, "/") {
		if segment != "" {
			segments = append(segments, segment)
		}
	}
	if len(segments) < 2 {
		return ""
	}
	finalID := strings.HasPrefix(segments[len(segments)-1], "{")
	resourceIndex := len(segments) - 1
	if finalID {
		resourceIndex--
	}
	parentIDIndex := resourceIndex - 1
	if parentIDIndex < 0 || !strings.HasPrefix(segments[parentIDIndex], "{") {
		return ""
	}
	resource := segments[resourceIndex]
	resources := map[string]string{"devicegroups": "device-group", "operationlists": "operation-list", "pipelines": "pipeline", "runs": "pipeline-run", "stageruns": "stage-run", "stages": "stage", "targetlists": "target-list", "targetruns": "target-run", "targets": "target"}
	return resources[resource]
}
