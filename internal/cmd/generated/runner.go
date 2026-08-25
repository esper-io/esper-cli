package generated

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
	"github.com/spf13/cobra"
)

type Operation struct {
	Generation   string
	Command      []string
	Method       string
	Path         string
	Noun         string
	Verb         string
	Pagination   string
	Destructive  bool
	ScopeParent  string
	Summary      string
	OperationID  string
	AliasOf      string
	SuccessMedia string
	Parameters   []Parameter
	Body         *Body
}
type Parameter struct {
	Name, In, Type  string
	ScopeName       string
	Required, Scope bool
	Enum            []string
}
type Body struct {
	MediaType  string
	Required   bool
	Empty      bool
	BodyOnly   bool
	Properties []Property
	AutoFill   []AutoFill
}
type AutoFill struct{ Name, Parameter, Type, Format string }
type Property struct {
	Name, Type, Format string
	Required, File     bool
	Enum               []string
}

var generatedOperations []Operation

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
			child = &cobra.Command{Use: segment, Short: commandGroupSummary(segment)}
			parent.AddCommand(child)
		}
		parent = child
	}
	verb := commandPath[len(commandPath)-1]
	summary := operations[0].Summary
	command := &cobra.Command{Use: verb, Short: summary, Args: cobra.ArbitraryArgs, RunE: func(command *cobra.Command, args []string) error {
		return run(command, args, operations, options)
	}}
	addFlags(command, operations)
	parent.AddCommand(command)
}

func commandGroupSummary(segment string) string {
	return fmt.Sprintf("Commands in the %s group", segment)
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

func addStringFlag(command *cobra.Command, name string, required bool, values []string, description string, seen map[string]bool) {
	name = kebab(name)
	if seen[name] {
		return
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
	operation, err := selectOperation(command, operations)
	if err != nil {
		return err
	}
	if options.JSON && !isJSONMedia(operation.SuccessMedia) {
		return esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("--json is not supported for %s responses", operation.SuccessMedia))
	}
	if err := validateRequiredParameters(command, operation); err != nil {
		return err
	}
	requestPath, err := replacePath(command, operation, args)
	if err != nil {
		return err
	}
	pathValues, err := resolvedPathValues(command, operation, args)
	if err != nil {
		return err
	}
	body, contentType, err := bodyForValues(command, operation, pathValues)
	if err != nil {
		return err
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
	store, err := esperruntime.NewStateStore()
	if err != nil {
		return esperruntime.NewError(esperruntime.CategoryAuth, err)
	}
	state, err := store.Load()
	if err != nil {
		return esperruntime.NewError(esperruntime.CategoryAuth, err)
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
	client := esperruntime.NewHTTPClient(credentials)
	response, err := client.DoWithContentTypeAndHeaders(command.Context(), operation.Method, requestPath, query, headers, body, contentType, operation.SuccessMedia)
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
	return esperruntime.WriteHuman(command.OutOrStdout(), response)
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
	return nil
}

func allPages(command *cobra.Command, client *esperruntime.HTTPClient, operation Operation, response []byte) ([]byte, error) {
	if operation.Pagination == "none" {
		return response, nil
	}
	var results []json.RawMessage
	for {
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

func replacePath(command *cobra.Command, operation Operation, args []string) (string, error) {
	values, err := resolvedPathValues(command, operation, args)
	if err != nil {
		return "", err
	}
	result := operation.Path
	for _, parameter := range operation.Parameters {
		if parameter.In != "path" {
			continue
		}
		name := "{" + parameter.Name + "}"
		result = strings.ReplaceAll(result, name, values[parameter.Name])
	}
	return result, nil
}

func resolvedPathValues(command *cobra.Command, operation Operation, args []string) (map[string]string, error) {
	values := map[string]string{}
	index := 0
	for _, parameter := range operation.Parameters {
		if parameter.In != "path" {
			continue
		}
		if parameter.Scope {
			value := flagString(command, parameterFlagName(parameter))
			if value == "" {
				return nil, esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("missing path parameter %s", parameter.Name))
			}
			values[parameter.Name] = value
			continue
		}
		if index >= len(args) {
			return nil, esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("missing path parameter %s", parameter.Name))
		}
		values[parameter.Name] = args[index]
		index++
	}
	if index != len(args) {
		return nil, esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("unexpected positional arguments"))
	}
	return values, nil
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
		autoValues = bodyAutoFill(operation, pathValues)
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
		values[property.Name] = typedValue(flagString(command, property.Name), property.Type)
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

func bodyAutoFill(operation Operation, pathValues map[string]string) map[string]any {
	values := map[string]any{}
	for _, fill := range operation.Body.AutoFill {
		if value, ok := pathValues[fill.Parameter]; ok {
			if isURLFormat(fill.Format) && operation.Body.MediaType == "application/json" {
				value = resourcePathForParameter(operation.Path, fill.Parameter, pathValues)
			}
			values[fill.Name] = typedValue(value, fill.Type)
		}
	}
	return values
}

func resourcePathForParameter(operationPath, parameter string, pathValues map[string]string) string {
	marker := "{" + parameter + "}"
	index := strings.Index(operationPath, marker)
	if index < 0 {
		return pathValues[parameter]
	}
	path := operationPath[:index+len(marker)]
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	for name, value := range pathValues {
		path = strings.ReplaceAll(path, "{"+name+"}", value)
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
func typedValue(value, kind string) any {
	if kind == "boolean" {
		return value == "true"
	}
	if kind == "integer" {
		var number int64
		_, _ = fmt.Sscan(value, &number)
		return number
	}
	if kind == "number" {
		var number float64
		_, _ = fmt.Sscan(value, &number)
		return number
	}
	return value
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
