package generated

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
	"github.com/spf13/cobra"
)

type Operation struct {
	Generation       string
	Command          []string
	Method           string
	Path             string
	Noun             string
	Verb             string
	Pagination       string
	ResponseEnvelope string
	RequireOneOf     []string
	Destructive      bool
	ScopeParent      string
	Summary          string
	Description      string
	Tags             []string
	DocsSlugs        []string
	OperationID      string
	AliasOf          string
	SuccessMedia     string
	Parameters       []Parameter
	Body             *Body
}
type Parameter struct {
	Name, In, Type, Description string
	ScopeName                   string
	Required, Scope             bool
	Enum                        []string
}
type Body struct {
	MediaType  string
	Required   bool
	Empty      bool
	BodyOnly   bool
	Properties []Property
	Fields     []BodyField
	AutoFill   []AutoFill
}
type AutoFill struct{ Name, Parameter, Type, Format string }
type Property struct {
	Name, Type, Format, Description string
	Example                         any
	Required, File                  bool
	Enum                            []string
}
type BodyField struct {
	Path, Type, Format, Description string
	Example                         any
	Required                        bool
	Enum                            []string
}

var generatedOperations []Operation

const maximumPaginationPages = 10000

const usageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

Available Commands:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

func UsageTemplate(heading string) string {
	return strings.Replace(usageTemplate, "Available Commands:", heading+":", 1)
}

func GroupSummary(segment string) string {
	if segment == "api" {
		return "Access older API generations"
	}
	return "Manage " + strings.ReplaceAll(segment, "-", " ") + " resources"
}

func init() {
	if err := json.Unmarshal(generatedOperationsJSON, &generatedOperations); err != nil {
		panic(err)
	}
}

func Operations() []Operation {
	return append([]Operation(nil), generatedOperations...)
}

func AddCommands(root *cobra.Command, options *esperruntime.GlobalOptions) {
	groups := executableOperationGroups(generatedOperations)
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		addCommand(root, strings.Split(key, "\x00"), groups[key], options)
	}
}

func executableOperationGroups(operations []Operation) map[string][]Operation {
	groups := map[string][]Operation{}
	for _, operation := range operations {
		if operation.AliasOf != "" {
			continue
		}
		groups[strings.Join(operation.Command, "\x00")] = append(groups[strings.Join(operation.Command, "\x00")], operation)
	}
	return groups
}

func addCommand(root *cobra.Command, commandPath []string, operations []Operation, options *esperruntime.GlobalOptions) {
	parent := root
	for _, segment := range commandPath[:len(commandPath)-1] {
		child, _, err := parent.Find([]string{segment})
		if err != nil || child == parent {
			child = &cobra.Command{Use: segment, Short: GroupSummary(segment)}
			if segment == "api" {
				child.SetUsageTemplate(UsageTemplate("API generations"))
			} else {
				child.SetUsageTemplate(UsageTemplate("Operations"))
			}
			parent.AddCommand(child)
		}
		parent = child
	}
	verb := commandPath[len(commandPath)-1]
	summary := operations[0].Summary
	command := &cobra.Command{Use: commandUse(verb, operations), Short: summary, Long: commandLongHelp(summary, operations), Args: cobra.ArbitraryArgs, RunE: func(command *cobra.Command, args []string) error {
		return run(command, args, operations, options)
	}}
	addFlags(command, operations)
	parent.AddCommand(command)
}

func commandLongHelp(summary string, operations []Operation) string {
	sections := []string{summary}
	if bodyHelp := bodySchemaHelp(operations); bodyHelp != "" {
		sections = append(sections, bodyHelp)
	}
	if len(operations) > 0 && !strings.HasPrefix(strings.Join(operations[0].Command, " "), "api ") {
		paths := map[string]bool{}
		for _, candidate := range generatedOperations {
			if candidate.AliasOf != "" || candidate.Noun != operations[0].Noun || candidate.Verb != operations[0].Verb {
				continue
			}
			path := strings.Join(candidate.Command, " ")
			if strings.HasPrefix(path, "api ") {
				paths[path] = true
			}
		}
		if len(paths) > 0 {
			older := mapKeys(paths)
			sort.Strings(older)
			var output strings.Builder
			output.WriteString("Other API generations:\n")
			for _, path := range older {
				output.WriteString("  espercli ")
				output.WriteString(path)
				output.WriteByte('\n')
			}
			sections = append(sections, strings.TrimSuffix(output.String(), "\n"))
		}
	}
	return strings.Join(sections, "\n\n")
}

func bodySchemaHelp(operations []Operation) string {
	fields := map[string]BodyField{}
	for _, operation := range operations {
		if operation.Body == nil {
			continue
		}
		for _, field := range operation.Body.Fields {
			if _, exists := fields[field.Path]; !exists {
				fields[field.Path] = field
			}
		}
	}
	if len(fields) == 0 {
		return ""
	}
	paths := make([]string, 0, len(fields))
	for path := range fields {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	var output strings.Builder
	output.WriteString("Request body fields (nested values require --body):\n")
	for _, path := range paths {
		field := fields[path]
		output.WriteString("  ")
		output.WriteString(path)
		output.WriteString(" (")
		output.WriteString(field.Type)
		if field.Format != "" {
			output.WriteString(", ")
			output.WriteString(field.Format)
		}
		if field.Required {
			output.WriteString(", required")
		}
		output.WriteString(")")
		if field.Description != "" {
			output.WriteString(": ")
			output.WriteString(field.Description)
		}
		if len(field.Enum) > 0 {
			output.WriteString(" [values: ")
			output.WriteString(strings.Join(field.Enum, ", "))
			output.WriteString("]")
		}
		if field.Example != nil {
			if example, err := json.Marshal(field.Example); err == nil {
				output.WriteString(" [example: ")
				output.Write(example)
				output.WriteByte(']')
			}
		}
		output.WriteByte('\n')
	}
	return strings.TrimSuffix(output.String(), "\n")
}

func commandUse(verb string, operations []Operation) string {
	if len(operations) == 0 {
		return verb
	}
	var parameters []string
	for _, parameter := range pathParameters(operations[0]) {
		if !parameter.Scope {
			parameters = append(parameters, "<"+kebab(parameter.Name)+">")
		}
	}
	if len(parameters) == 0 {
		return verb
	}
	return verb + " " + strings.Join(parameters, " ")
}

func addFlags(command *cobra.Command, operations []Operation) {
	seen := map[string]bool{}
	for _, operation := range operations {
		for _, parameter := range operation.Parameters {
			if parameter.In == "path" && !parameter.Scope {
				continue
			}
			required := false
			description := parameterFlagName(parameter)
			if contains(operation.RequireOneOf, parameter.Name) {
				description += " (at least one required from: " + formatFlagNames(operation.RequireOneOf) + ")"
			} else if parameter.Required && parameterRequiredForAllOperations(operations, parameter) {
				description += " (required)"
			}
			if parameter.Scope {
				description = "scope routes to " + operation.Path
			}
			addStringFlag(command, parameterFlagName(parameter), required, parameter.Enum, description, seen)
		}
		if operation.Pagination == "limit-offset" || operation.Pagination == "apps-envelope" {
			addStringFlag(command, "limit", false, nil, "limit", seen)
			addStringFlag(command, "offset", false, nil, "offset", seen)
			if !seen["all"] {
				command.Flags().Bool("all", false, "fetch all result pages")
				seen["all"] = true
			}
		}
		if !isJSONMedia(operation.SuccessMedia) {
			addStringFlag(command, "output", false, nil, "write raw response to path", seen)
		}
		if operation.Body == nil {
			continue
		}
		if operation.Body.MediaType == "application/json" {
			description := "inline JSON, @path, or - for stdin"
			if operation.Body.BodyOnly {
				description += " (required)"
			}
			addStringFlag(command, "body", false, nil, description, seen)
		}
		for _, property := range operation.Body.Properties {
			addStringFlag(command, property.Name, false, property.Enum, property.Name, seen)
		}
	}
}

func parameterRequiredForAllOperations(operations []Operation, wanted Parameter) bool {
	for _, operation := range operations {
		found := false
		for _, parameter := range operation.Parameters {
			if parameter.Name == wanted.Name && parameter.In == wanted.In && parameter.Required {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return len(operations) > 0
}

func addStringFlag(command *cobra.Command, name string, required bool, values []string, description string, seen map[string]bool) {
	name = kebab(name)
	if seen[name] {
		return
	}
	if len(values) > 0 {
		description += " (values: " + strings.Join(values, ", ") + ")"
	}
	command.Flags().String(name, "", description)
	seen[name] = true
	if required {
		_ = command.MarkFlagRequired(name)
	}
	if len(values) > 0 {
		_ = command.RegisterFlagCompletionFunc(name, func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return values, cobra.ShellCompDirectiveDefault
		})
	}
}

func run(command *cobra.Command, args []string, operations []Operation, options *esperruntime.GlobalOptions) error {
	store, err := esperruntime.NewStateStore()
	if err != nil {
		return esperruntime.NewError(esperruntime.CategoryAuth, err)
	}
	state, err := store.Load()
	if err != nil {
		return esperruntime.NewError(esperruntime.CategoryAuth, err)
	}
	if err := applyScopeContextFallbacks(command, operations, state.Active, options.Verbose); err != nil {
		return err
	}
	operation, err := selectOperation(command, operations)
	if err != nil {
		return addScopeContextHint(command, operations, state.Active, err)
	}
	if options.JSON && !isJSONMedia(operation.SuccessMedia) {
		return esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("--json is not supported for %s responses", operation.SuccessMedia))
	}
	if err := validateRequiredParameters(command, operation); err != nil {
		return err
	}
	pathValues, err := resolvedPathValuesWithContext(command, operation, args, state.Active, options.Verbose)
	if err != nil {
		return err
	}
	requestPath := replacePathValues(operation, pathValues)
	body, contentType, err := bodyForValues(command, operation, pathValues)
	if err != nil {
		return err
	}
	credentials, err := esperruntime.ResolveCredentials(state.Config, options.Environment, options.APIKey)
	if err != nil {
		return err
	}
	body, err = qualifyAutoFillURLs(body, operation.Body, credentials.BaseURL())
	if err != nil {
		return err
	}
	query := make(map[string][]string)
	headers := make(map[string][]string)
	for _, parameter := range operation.Parameters {
		if parameter.In == "query" && command.Flags().Changed(kebab(parameter.Name)) {
			query[parameter.Name] = []string{flagString(command, parameter.Name)}
		}
		if parameter.In == "header" && command.Flags().Changed(kebab(parameter.Name)) {
			headers[parameter.Name] = []string{flagString(command, parameter.Name)}
		}
	}
	if operation.Pagination == "limit-offset" || operation.Pagination == "apps-envelope" {
		for _, name := range []string{"limit", "offset"} {
			if command.Flags().Changed(name) {
				query[name] = []string{flagString(command, name)}
			}
		}
	}
	if isWriteOperation(operation) {
		request, approved, err := esperruntime.RequireApproval(esperruntime.ApprovalSpec{BaseURL: credentials.BaseURL(), Method: operation.Method, Path: requestPath, ContentType: contentType, Query: query, Headers: headers, Body: body})
		if err != nil {
			return esperruntime.NewError(esperruntime.CategoryAuth, err)
		}
		if !approved {
			return esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("approval required for %s %s\nreview: espercli approval show %s\nhuman approval: espercli approval approve %s", operation.Method, requestPath, request.ID, request.ID))
		}
	}
	if operation.Destructive {
		ok, err := esperruntime.Confirm(command.InOrStdin(), command.ErrOrStderr(), requestPath, 1, options.Yes)
		if err != nil {
			return err
		}
		if !ok {
			return esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("operation cancelled"))
		}
	}
	client := esperruntime.NewHTTPClient(credentials)
	accept := operation.SuccessMedia
	if accept == "" {
		accept = "application/json"
	}
	response, err := client.DoWithContentTypeAndHeaders(command.Context(), operation.Method, requestPath, query, headers, body, contentType, accept)
	if err != nil {
		return err
	}
	all, _ := command.Flags().GetBool("all")
	if all {
		response, err = allPages(command, client, operation, response)
		if err != nil {
			return err
		}
	}
	if !isJSONMedia(operation.SuccessMedia) {
		return writeRawResponse(command, response)
	}
	if options.JSON {
		return esperruntime.WriteJSON(command.OutOrStdout(), response)
	}
	response, err = esperruntime.UnwrapResponseEnvelope(response, operation.ResponseEnvelope)
	if err != nil {
		return esperruntime.NewError(esperruntime.CategoryAPI, err)
	}
	return esperruntime.WriteHuman(command.OutOrStdout(), response)
}

func isWriteOperation(operation Operation) bool {
	return operation.Method == "POST" || operation.Method == "PUT" || operation.Method == "PATCH" || operation.Method == "DELETE"
}

func writeRawResponse(command *cobra.Command, response []byte) error {
	if command.Flags().Changed("output") {
		if err := os.WriteFile(flagString(command, "output"), response, 0o644); err != nil {
			return fmt.Errorf("write --output: %w", err)
		}
		return nil
	}
	_, err := command.OutOrStdout().Write(response)
	return err
}

func validateRequiredParameters(command *cobra.Command, operation Operation) error {
	for _, parameter := range operation.Parameters {
		if !parameter.Required || (parameter.In != "query" && parameter.In != "header") {
			continue
		}
		if !command.Flags().Changed(kebab(parameter.Name)) {
			return esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("required flag --%s is not set", kebab(parameter.Name)))
		}
	}
	if len(operation.RequireOneOf) > 0 {
		for _, name := range operation.RequireOneOf {
			if command.Flags().Changed(kebab(name)) {
				return nil
			}
		}
		return esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("at least one of %s is required", formatFlagNames(operation.RequireOneOf)))
	}
	return nil
}

func formatFlagNames(names []string) string {
	flags := make([]string, 0, len(names))
	for _, name := range names {
		flags = append(flags, "--"+kebab(name))
	}
	return strings.Join(flags, ", ")
}

func allPages(command *cobra.Command, client *esperruntime.HTTPClient, operation Operation, response []byte) ([]byte, error) {
	return allPagesWithLimit(command, client, operation, response, maximumPaginationPages)
}

func allPagesWithLimit(command *cobra.Command, client *esperruntime.HTTPClient, operation Operation, response []byte, limit int) ([]byte, error) {
	if operation.Pagination == "none" {
		return response, nil
	}
	var results []json.RawMessage
	seenNext := map[string]struct{}{}
	for pageCount := 0; ; pageCount++ {
		if pageCount >= limit {
			return nil, esperruntime.NewError(esperruntime.CategoryAPI, fmt.Errorf("pagination exceeded %d pages", limit))
		}
		page, err := unwrapPage(operation.Pagination, response)
		if err != nil {
			return nil, esperruntime.NewError(esperruntime.CategoryAPI, err)
		}
		results = append(results, page.Results...)
		if page.Next == "" {
			return esperruntime.MarshalMergedResults(results)
		}
		next, err := url.Parse(page.Next)
		if err != nil || next.Path == "" {
			return nil, esperruntime.NewError(esperruntime.CategoryAPI, fmt.Errorf("invalid pagination next URL %q", page.Next))
		}
		nextKey := next.EscapedPath()
		if query := next.Query().Encode(); query != "" {
			nextKey += "?" + query
		}
		if _, seen := seenNext[nextKey]; seen {
			return nil, esperruntime.NewError(esperruntime.CategoryAPI, fmt.Errorf("pagination next URL repeated: %q", page.Next))
		}
		seenNext[nextKey] = struct{}{}
		response, err = client.DoWithContentType(command.Context(), operation.Method, next.EscapedPath(), next.Query(), nil, "application/json")
		if err != nil {
			return nil, err
		}
	}
}

func unwrapPage(kind string, response []byte) (esperruntime.Page, error) {
	if kind == "apps-envelope" {
		return esperruntime.UnwrapAppsEnvelope(response)
	}
	return esperruntime.UnwrapLimitOffset(response)
}

func selectOperation(command *cobra.Command, operations []Operation) (Operation, error) {
	validScopes := map[string]bool{}
	for _, operation := range operations {
		for _, scope := range operationScopeNames(operation) {
			validScopes[scope] = true
		}
	}
	selectedScopes := map[string]bool{}
	for scope := range validScopes {
		if command.Flags().Changed(kebab(scope)) {
			selectedScopes[scope] = true
		}
	}
	var selected []Operation
	for _, operation := range operations {
		if sameScopes(operationScopeNames(operation), selectedScopes) {
			selected = append(selected, operation)
		}
	}
	if len(selected) == 1 {
		return selected[0], nil
	}
	combinations := make([]string, 0, len(operations))
	for _, operation := range operations {
		names := operationScopeNames(operation)
		if len(names) == 0 {
			combinations = append(combinations, "no scope flags")
			continue
		}
		flags := make([]string, 0, len(names))
		for _, name := range names {
			flags = append(flags, "--"+kebab(name))
		}
		combinations = append(combinations, strings.Join(flags, " + "))
	}
	sort.Strings(combinations)
	return Operation{}, esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("scope flags must match one route: %s", strings.Join(combinations, ", ")))
}

func operationScopeNames(operation Operation) []string {
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

func sameScopes(names []string, selected map[string]bool) bool {
	if len(names) != len(selected) {
		return false
	}
	for _, name := range names {
		if !selected[name] {
			return false
		}
	}
	return true
}

func applyScopeContextFallbacks(command *cobra.Command, operations []Operation, active esperruntime.ActiveContext, verbose bool) error {
	selectedScopes := selectedScopeNames(command, operations)
	for _, operation := range operations {
		if sameScopes(operationScopeNames(operation), selectedScopes) {
			return nil
		}
	}

	type candidate struct {
		values map[string]contextValue
	}
	var candidates []candidate
	for _, operation := range operations {
		values := map[string]contextValue{}
		possible := true
		for _, selected := range mapKeys(selectedScopes) {
			if !contains(operationScopeNames(operation), selected) {
				possible = false
				break
			}
		}
		if !possible {
			continue
		}
		for _, scope := range operationScopeNames(operation) {
			if selectedScopes[scope] {
				continue
			}
			parameter, ok := scopeParameter(operation, scope)
			if !ok {
				possible = false
				break
			}
			resource, ok := esperruntime.ContextResourceForParameter(parameter.Name)
			if !ok || active.Resource(resource) == nil || active.Resource(resource).ID == "" {
				possible = false
				break
			}
			values[scope] = contextValue{parameter: parameter.Name, resource: resource, value: active.Resource(resource).ID}
		}
		if possible && len(values) > 0 {
			candidates = append(candidates, candidate{values: values})
		}
	}
	if len(candidates) != 1 {
		return nil
	}
	for _, scope := range sortedContextValueKeys(candidates[0].values) {
		value := candidates[0].values[scope]
		if err := command.Flags().Set(kebab(scope), value.value); err != nil {
			return esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("set context fallback --%s: %w", kebab(scope), err))
		}
		writeContextDiagnostic(command, verbose, value)
	}
	return nil
}

type contextValue struct {
	parameter string
	resource  string
	value     string
}

func selectedScopeNames(command *cobra.Command, operations []Operation) map[string]bool {
	selected := map[string]bool{}
	for _, operation := range operations {
		for _, scope := range operationScopeNames(operation) {
			if command.Flags().Changed(kebab(scope)) {
				selected[scope] = true
			}
		}
	}
	return selected
}

func scopeParameter(operation Operation, scope string) (Parameter, bool) {
	for _, parameter := range operation.Parameters {
		if parameter.In == "path" && parameter.Scope && parameterFlagName(parameter) == kebab(scope) {
			return parameter, true
		}
	}
	return Parameter{}, false
}

func addScopeContextHint(command *cobra.Command, operations []Operation, active esperruntime.ActiveContext, selectionError error) error {
	resources := map[string]bool{}
	for _, operation := range operations {
		for _, parameter := range operation.Parameters {
			if parameter.In != "path" || !parameter.Scope || command.Flags().Changed(parameterFlagName(parameter)) {
				continue
			}
			resource, ok := esperruntime.ContextResourceForParameter(parameter.Name)
			if ok && (active.Resource(resource) == nil || active.Resource(resource).ID == "") {
				resources[resource] = true
			}
		}
	}
	if len(resources) == 0 {
		return selectionError
	}
	names := mapKeys(resources)
	sort.Strings(names)
	hints := make([]string, 0, len(names))
	for _, resource := range names {
		hints = append(hints, fmt.Sprintf("espercli context set %s <id>", resource))
	}
	return esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("%s; alternatively run %s", selectionError, strings.Join(hints, " or ")))
}

func replacePath(command *cobra.Command, operation Operation, args []string) (string, error) {
	values, err := resolvedPathValues(command, operation, args)
	if err != nil {
		return "", err
	}
	return replacePathValues(operation, values), nil
}

func replacePathValues(operation Operation, values map[string]string) string {
	result := operation.Path
	for _, parameter := range operation.Parameters {
		if parameter.In != "path" {
			continue
		}
		name := "{" + parameter.Name + "}"
		result = strings.ReplaceAll(result, name, url.PathEscape(values[parameter.Name]))
	}
	return result
}

func resolvedPathValues(command *cobra.Command, operation Operation, args []string) (map[string]string, error) {
	return resolvedPathValuesWithContext(command, operation, args, esperruntime.ActiveContext{}, false)
}

func resolvedPathValuesWithContext(command *cobra.Command, operation Operation, args []string, active esperruntime.ActiveContext, verbose bool) (map[string]string, error) {
	values := map[string]string{}
	index := 0
	for _, parameter := range pathParameters(operation) {
		if parameter.Scope {
			value := flagString(command, parameterFlagName(parameter))
			if value == "" {
				return nil, missingPathParameterError(parameter.Name)
			}
			values[parameter.Name] = value
			continue
		}
		if index >= len(args) {
			resource, ok := esperruntime.ContextResourceForParameter(parameter.Name)
			if !ok || active.Resource(resource) == nil || active.Resource(resource).ID == "" {
				return nil, missingPathParameterError(parameter.Name)
			}
			value := contextValue{parameter: parameter.Name, resource: resource, value: active.Resource(resource).ID}
			values[parameter.Name] = value.value
			writeContextDiagnostic(command, verbose, value)
			continue
		}
		values[parameter.Name] = args[index]
		index++
	}
	if index != len(args) {
		return nil, esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("unexpected positional arguments"))
	}
	return values, nil
}

func pathParameters(operation Operation) []Parameter {
	var parameters []Parameter
	for _, parameter := range operation.Parameters {
		if parameter.In == "path" {
			parameters = append(parameters, parameter)
		}
	}
	sort.SliceStable(parameters, func(left, right int) bool {
		leftIndex := strings.Index(operation.Path, "{"+parameters[left].Name+"}")
		rightIndex := strings.Index(operation.Path, "{"+parameters[right].Name+"}")
		if leftIndex < 0 {
			leftIndex = len(operation.Path)
		}
		if rightIndex < 0 {
			rightIndex = len(operation.Path)
		}
		return leftIndex < rightIndex
	})
	return parameters
}

func missingPathParameterError(parameter string) error {
	resource, ok := esperruntime.ContextResourceForParameter(parameter)
	if !ok {
		return esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("missing path parameter %s", parameter))
	}
	return esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("missing path parameter %s (or run espercli context set %s <id>)", parameter, resource))
}

func writeContextDiagnostic(command *cobra.Command, verbose bool, value contextValue) {
	if verbose {
		_, _ = fmt.Fprintf(command.ErrOrStderr(), "context: using active %s %s for %s\n", value.resource, value.value, value.parameter)
	}
}

func sortedContextValueKeys(values map[string]contextValue) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func mapKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func bodyFor(command *cobra.Command, operation Operation) ([]byte, string, error) {
	return bodyForValues(command, operation, nil)
}

func bodyForValues(command *cobra.Command, operation Operation, pathValues map[string]string) ([]byte, string, error) {
	if operation.Body == nil {
		return nil, "application/json", nil
	}
	bodyFlag := command.Flags().Changed("body")
	propertiesSet := false
	for _, property := range operation.Body.Properties {
		propertiesSet = propertiesSet || command.Flags().Changed(kebab(property.Name))
	}
	if bodyFlag && propertiesSet {
		return nil, "", esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("--body cannot be combined with property flags"))
	}
	if operation.Body.BodyOnly && !bodyFlag {
		return nil, "", esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("--body is required"))
	}
	autoValues := map[string]any{}
	// Optional bodies remain absent unless the caller supplies a body input.
	if operation.Body.Required || bodyFlag || propertiesSet {
		var err error
		autoValues, err = bodyAutoFill(operation, pathValues)
		if err != nil {
			return nil, "", err
		}
	}
	if operation.Body.Required && !operation.Body.Empty && !bodyFlag && !propertiesSet && len(autoValues) == 0 {
		return nil, "", esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("required request body needs at least one input: %s", strings.Join(bodyInputFlags(operation.Body), ", ")))
	}
	if operation.Body.MediaType == "application/json" && bodyFlag {
		data, err := readJSONBody(flagString(command, "body"), command.InOrStdin())
		if err != nil {
			return nil, "", err
		}
		return mergeAutoFillJSON(data, autoValues)
	}
	if operation.Body.MediaType == "multipart/form-data" {
		if !operation.Body.Required && !propertiesSet && len(autoValues) == 0 {
			return nil, "multipart/form-data", nil
		}
		return multipartBody(command, operation.Body, autoValues)
	}
	values := map[string]any{}
	for name, value := range autoValues {
		values[name] = value
	}
	for _, property := range operation.Body.Properties {
		if !command.Flags().Changed(kebab(property.Name)) {
			continue
		}
		value, err := typedValue(flagString(command, property.Name), property.Type)
		if err != nil {
			return nil, "", esperruntime.NewError(esperruntime.CategoryUsage, err)
		}
		values[property.Name] = value
	}
	if operation.Body.Required || propertiesSet {
		for _, property := range operation.Body.Properties {
			if property.Required && !command.Flags().Changed(kebab(property.Name)) {
				return nil, "", esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("required flag --%s is not set", kebab(property.Name)))
			}
		}
	}
	if len(values) == 0 && !operation.Body.Required {
		return nil, "application/json", nil
	}
	if operation.Body.Empty {
		return []byte("{}"), "application/json", nil
	}
	data, err := json.Marshal(values)
	return data, "application/json", err
}

func bodyInputFlags(body *Body) []string {
	var flags []string
	if body.MediaType == "application/json" {
		flags = append(flags, "--body")
	}
	for _, property := range body.Properties {
		flags = append(flags, "--"+kebab(property.Name))
	}
	return flags
}

func bodyAutoFill(operation Operation, pathValues map[string]string) (map[string]any, error) {
	values := map[string]any{}
	for _, fill := range operation.Body.AutoFill {
		if value, ok := pathValues[fill.Parameter]; ok {
			if isURLFormat(fill.Format) && operation.Body.MediaType == "application/json" {
				value = resourcePathForParameter(operation.Path, fill.Parameter, pathValues)
			}
			typed, err := typedValue(value, fill.Type)
			if err != nil {
				return nil, esperruntime.NewError(esperruntime.CategoryUsage, err)
			}
			values[fill.Name] = typed
		}
	}
	return values, nil
}

func resourcePathForParameter(operationPath, parameter string, pathValues map[string]string) string {
	marker := "{" + parameter + "}"
	index := strings.Index(operationPath, marker)
	if index < 0 {
		return url.PathEscape(pathValues[parameter])
	}
	path := operationPath[:index+len(marker)]
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	for name, value := range pathValues {
		path = strings.ReplaceAll(path, "{"+name+"}", url.PathEscape(value))
	}
	return path
}

func qualifyAutoFillURLs(data []byte, body *Body, baseURL string) ([]byte, error) {
	if len(data) == 0 || body == nil || body.MediaType != "application/json" {
		return data, nil
	}
	names := map[string]bool{}
	for _, fill := range body.AutoFill {
		if isURLFormat(fill.Format) {
			names[fill.Name] = true
		}
	}
	if len(names) == 0 {
		return data, nil
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil || object == nil {
		return nil, esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("URL-valued path auto-fill requires a JSON object"))
	}
	for name := range names {
		value, ok := object[name].(string)
		if ok && strings.HasPrefix(value, "/") {
			object[name] = strings.TrimRight(baseURL, "/") + value
		}
	}
	return json.Marshal(object)
}

func isURLFormat(format string) bool {
	return format == "url" || format == "uri"
}

func mergeAutoFillJSON(data []byte, values map[string]any) ([]byte, string, error) {
	if len(values) == 0 {
		return data, "application/json", nil
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil || object == nil {
		return nil, "", esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("--body must be a JSON object when path values are auto-filled"))
	}
	for name, value := range values {
		object[name] = value
	}
	result, err := json.Marshal(object)
	return result, "application/json", err
}

func multipartBody(command *cobra.Command, body *Body, autoValues map[string]any) ([]byte, string, error) {
	var output strings.Builder
	writer := multipart.NewWriter(&output)
	for name, value := range autoValues {
		if err := writer.WriteField(name, fmt.Sprint(value)); err != nil {
			return nil, "", err
		}
	}
	for _, property := range body.Properties {
		flag := kebab(property.Name)
		if !command.Flags().Changed(flag) {
			if property.Required {
				return nil, "", esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("required flag --%s is not set", flag))
			}
			continue
		}
		value := flagString(command, property.Name)
		if property.File {
			file, err := os.Open(value)
			if err != nil {
				return nil, "", esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("open --%s: %w", flag, err))
			}
			part, err := writer.CreateFormFile(property.Name, filepath.Base(value))
			if err == nil {
				_, err = io.Copy(part, file)
			}
			file.Close()
			if err != nil {
				return nil, "", fmt.Errorf("write --%s: %w", flag, err)
			}
			continue
		}
		if err := writer.WriteField(property.Name, value); err != nil {
			return nil, "", err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return []byte(output.String()), writer.FormDataContentType(), nil
}

func readJSONBody(value string, input io.Reader) ([]byte, error) {
	var data []byte
	var err error
	if value == "-" {
		data, err = io.ReadAll(input)
	} else if strings.HasPrefix(value, "@") {
		data, err = os.ReadFile(strings.TrimPrefix(value, "@"))
	} else {
		data = []byte(value)
	}
	if err != nil {
		return nil, esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("read --body: %w", err))
	}
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("validate --body JSON: %w", err))
	}
	return data, nil
}
func typedValue(value, kind string) (any, error) {
	if kind == "boolean" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("invalid boolean value %q: %w", value, err)
		}
		return parsed, nil
	}
	if kind == "integer" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid integer value %q: %w", value, err)
		}
		return parsed, nil
	}
	if kind == "number" {
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number value %q: %w", value, err)
		}
		if math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			return nil, fmt.Errorf("invalid number value %q", value)
		}
		return parsed, nil
	}
	return value, nil
}
func flagString(command *cobra.Command, name string) string {
	value, _ := command.Flags().GetString(kebab(name))
	return value
}

func parameterFlagName(parameter Parameter) string {
	if parameter.Scope {
		if parameter.ScopeName != "" {
			return kebab(parameter.ScopeName)
		}
		name := strings.TrimSuffix(parameter.Name, "_id")
		name = strings.TrimSuffix(name, "Id")
		return kebab(name)
	}
	return kebab(parameter.Name)
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

func isJSONMedia(mediaType string) bool {
	return mediaType == "" || strings.Contains(mediaType, "/json") || strings.Contains(mediaType, "+json")
}
