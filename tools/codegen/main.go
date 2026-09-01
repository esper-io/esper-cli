package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var methods = map[string]bool{"get": true, "post": true, "put": true, "patch": true, "delete": true}
var coreGenerationRank = map[string]int{"legacy": 0, "v0": 1, "v1": 2, "v2": 3, "v3": 4}
var pathResourceNames = map[string]string{"alarmrules": "alarm-rule", "devicegroup": "device-group", "devicegroups": "device-group", "operationlists": "operation-list", "pipelines": "pipeline", "runs": "pipeline-run", "stageruns": "stage-run", "stages": "stage", "targetlists": "target-list", "targetruns": "target-run", "targets": "target"}
var sideFamilyPrefixes = map[string]string{"authn2": "authn", "authz2": "authz", "foundry": "foundry", "geofence": "geofence", "pipelines-v0": "pipeline"}

type document struct {
	Info struct {
		Generation string `json:"x-esper-generation"`
	} `json:"info"`
	Paths      map[string]map[string]operation `json:"paths"`
	Components struct {
		Schemas    map[string]schema    `json:"schemas"`
		Parameters map[string]parameter `json:"parameters"`
	} `json:"components"`
}

type operation struct {
	Parameters []parameter `json:"parameters"`
	Body       struct {
		Required bool                    `json:"required"`
		Content  map[string]requestMedia `json:"content"`
	} `json:"requestBody"`
	Responses    map[string]response `json:"responses"`
	OperationID  string              `json:"operationId"`
	AliasOf      string              `json:"x-esper-alias-of"`
	Noun         string              `json:"x-esper-noun"`
	Verb         string              `json:"x-esper-verb"`
	Pagination   string              `json:"x-esper-pagination"`
	Envelope     string              `json:"x-esper-response-envelope"`
	RequireOneOf []string            `json:"x-esper-require-one-of"`
	Destructive  bool                `json:"x-esper-destructive"`
	ScopeParent  string              `json:"x-esper-scope-parent"`
	Summary      string              `json:"summary"`
	Description  string              `json:"description"`
	Tags         []string            `json:"tags"`
	DocsSlugs    []string            `json:"x-esper-docs-slugs"`
}

type response struct {
	Content map[string]json.RawMessage `json:"content"`
}

type parameter struct {
	Ref         string `json:"$ref"`
	Name        string `json:"name"`
	In          string `json:"in"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Schema      schema `json:"schema"`
}

type requestMedia struct {
	Schema schema `json:"schema"`
}
type schema struct {
	Ref         string            `json:"$ref"`
	Type        any               `json:"type"`
	Format      string            `json:"format"`
	Description string            `json:"description"`
	Example     any               `json:"example"`
	Enum        []any             `json:"enum"`
	Items       *schema           `json:"items"`
	Properties  map[string]schema `json:"properties"`
	Required    []string          `json:"required"`
	AllOf       []schema          `json:"allOf"`
	AnyOf       []schema          `json:"anyOf"`
	OneOf       []schema          `json:"oneOf"`
}

type generatedOperation struct {
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
	Parameters       []generatedParameter
	Body             *generatedBody
}
type generatedParameter struct {
	Name, In, Type, Description string
	ScopeName                   string
	Required, Scope             bool
	Enum                        []string
}
type generatedBody struct {
	MediaType  string
	Required   bool
	Empty      bool
	BodyOnly   bool
	Properties []generatedProperty
	Fields     []generatedBodyField
	AutoFill   []generatedAutoFill
}
type generatedAutoFill struct{ Name, Parameter, Type, Format string }
type generatedProperty struct {
	Name, Type, Format, Description string
	Example                         any
	Required, File                  bool
	Enum                            []string
}
type generatedBodyField struct {
	Path, Type, Format, Description string
	Example                         any
	Required                        bool
	Enum                            []string
}

func main() {
	specDir := flag.String("spec-dir", "spec/openapi", "directory containing resolved OpenAPI files")
	output := flag.String("output", "internal/cmd/generated/zz_generated_commands.go", "generated Go source path")
	flag.Parse()
	operations, err := load(*specDir)
	if err != nil {
		fatal(err)
	}
	assignCommands(operations)
	sort.Slice(operations, func(i, j int) bool {
		left, right := strings.Join(operations[i].Command, "\x00"), strings.Join(operations[j].Command, "\x00")
		if left != right {
			return left < right
		}
		if operations[i].ScopeParent != operations[j].ScopeParent {
			return operations[i].ScopeParent < operations[j].ScopeParent
		}
		if operations[i].Path != operations[j].Path {
			return operations[i].Path < operations[j].Path
		}
		return operations[i].Method < operations[j].Method
	})
	data, err := json.Marshal(operations)
	if err != nil {
		fatal(err)
	}
	var source bytes.Buffer
	source.WriteString("// Code generated by tools/codegen; DO NOT EDIT.\n\npackage generated\n\n")
	source.WriteString("var generatedOperationsJSON = []byte(")
	source.WriteString(strconv.Quote(string(data)))
	source.WriteString(")\n")
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fatal(err)
	}
	if err := os.WriteFile(*output, source.Bytes(), 0o644); err != nil {
		fatal(err)
	}
}

func load(directory string) ([]generatedOperation, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	var result []generatedOperation
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			return nil, err
		}
		var spec document
		if err := json.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("decode %s: %w", entry.Name(), err)
		}
		if err := validateGeneration(spec.Info.Generation); err != nil {
			return nil, fmt.Errorf("validate %s: %w", entry.Name(), err)
		}
		for apiPath, item := range spec.Paths {
			for method, operation := range item {
				if !methods[method] {
					continue
				}
				generated := generatedOperation{Generation: spec.Info.Generation, Method: strings.ToUpper(method), Path: apiPath, Noun: operation.Noun, Verb: operation.Verb, Pagination: operation.Pagination, ResponseEnvelope: operation.Envelope, RequireOneOf: operation.RequireOneOf, Destructive: operation.Destructive, ScopeParent: operation.ScopeParent, Summary: operation.Summary, Description: operation.Description, Tags: operation.Tags, DocsSlugs: operation.DocsSlugs, OperationID: operation.OperationID, AliasOf: operation.AliasOf, SuccessMedia: successMedia(operation.Responses)}
				for _, parameter := range operation.Parameters {
					parameter = resolveParameter(parameter, spec.Components.Parameters)
					resolved := resolve(parameter.Schema, spec.Components.Schemas)
					scopeName := scopeParameterNames(apiPath, operation.ScopeParent)[parameter.Name]
					generated.Parameters = append(generated.Parameters, generatedParameter{Name: parameter.Name, In: parameter.In, Type: scalarType(resolved), Description: parameter.Description, Required: parameter.Required, Scope: scopeName != "", ScopeName: scopeName, Enum: stringEnum(resolved.Enum)})
				}
				if len(operation.Body.Content) > 0 {
					generated.Body = extractBody(operation.Body.Content, operation.Body.Required, generated.Parameters, spec.Components.Schemas)
				}
				result = append(result, generated)
			}
		}
	}
	deriveResponseEnvelopes(result)
	return result, nil
}

func deriveResponseEnvelopes(operations []generatedOperation) {
	families := map[string]bool{}
	for _, operation := range operations {
		if operation.Pagination == "apps-envelope" {
			families[serviceFamily(operation.Path)] = true
		}
	}
	for index := range operations {
		if operations[index].ResponseEnvelope == "" && families[serviceFamily(operations[index].Path)] {
			operations[index].ResponseEnvelope = "apps-envelope"
		}
	}
}

func serviceFamily(apiPath string) string {
	segments := strings.Split(strings.Trim(apiPath, "/"), "/")
	if len(segments) < 2 {
		return strings.Join(segments, "/")
	}
	return strings.Join(segments[:2], "/")
}

func scopeParameterNames(apiPath, scopeParent string) map[string]string {
	result := map[string]string{}
	if scopeParent == "" {
		return result
	}
	segments := strings.Split(strings.Trim(apiPath, "/"), "/")
	lastIsItem := strings.HasPrefix(segments[len(segments)-1], "{")
	for index := 0; index+1 < len(segments); index++ {
		parameter := segments[index+1]
		if !strings.HasPrefix(parameter, "{") || !strings.HasSuffix(parameter, "}") || (lastIsItem && index+1 == len(segments)-1) {
			continue
		}
		resource := pathResourceNames[segments[index]]
		if resource == "" {
			resource = strings.TrimSuffix(segments[index], "s")
		}
		result[strings.TrimSuffix(strings.TrimPrefix(parameter, "{"), "}")] = resource
	}
	return result
}

func resolve(value schema, schemas map[string]schema) schema {
	return resolveWithSeen(value, schemas, map[string]bool{})
}

func resolveWithSeen(value schema, schemas map[string]schema, resolving map[string]bool) schema {
	if !strings.HasPrefix(value.Ref, "#/components/schemas/") {
		return value
	}
	if resolving[value.Ref] {
		return value
	}
	name := strings.TrimPrefix(value.Ref, "#/components/schemas/")
	resolved, ok := schemas[name]
	if !ok {
		return value
	}
	resolving[value.Ref] = true
	defer delete(resolving, value.Ref)
	return resolveWithSeen(resolved, schemas, resolving)
}

func resolveParameter(value parameter, parameters map[string]parameter) parameter {
	if !strings.HasPrefix(value.Ref, "#/components/parameters/") {
		return value
	}
	name := strings.TrimPrefix(value.Ref, "#/components/parameters/")
	resolved, ok := parameters[name]
	if !ok {
		return value
	}
	return resolveParameter(resolved, parameters)
}

func extractBody(content map[string]requestMedia, required bool, parameters []generatedParameter, schemas map[string]schema) *generatedBody {
	mediaType := ""
	if _, ok := content["application/json"]; ok {
		mediaType = "application/json"
	} else if _, ok := content["multipart/form-data"]; ok {
		mediaType = "multipart/form-data"
	} else {
		return nil
	}
	value := resolve(content[mediaType].Schema, schemas)
	requiredProperties := map[string]bool{}
	for _, name := range value.Required {
		requiredProperties[name] = true
	}
	_, explicitlyObject := value.Type.(string)
	body := &generatedBody{MediaType: mediaType, Required: required, Empty: explicitlyObject && scalarType(value) == "object" && len(value.Properties) == 0}
	parameterByFlag := map[string]generatedParameter{}
	for _, parameter := range parameters {
		if parameter.In == "path" {
			parameterByFlag[kebab(parameter.Name)] = parameter
			parameterByFlag[kebab(parameter.ScopeName)] = parameter
		}
	}
	body.BodyOnly = hasRequiredComplexProperty(value, schemas, map[string]bool{})
	for _, name := range sortedKeys(value.Properties) {
		property := resolve(value.Properties[name], schemas)
		typeName := scalarType(property)
		file := mediaType == "multipart/form-data" && typeName == "string" && property.Format == "binary"
		if parameter, collision := parameterByFlag[kebab(name)]; collision {
			if requiredProperties[name] {
				body.AutoFill = append(body.AutoFill, generatedAutoFill{Name: name, Parameter: parameter.Name, Type: typeName, Format: property.Format})
			}
			continue
		}
		if body.BodyOnly {
			continue
		}
		if !file && !isScalar(typeName) {
			continue
		}
		body.Properties = append(body.Properties, generatedProperty{Name: name, Type: typeName, Format: property.Format, Description: property.Description, Example: property.Example, Required: requiredProperties[name], File: file, Enum: stringEnum(property.Enum)})
	}
	body.Fields = bodyFields(value, schemas)
	return body
}

func bodyFields(value schema, schemas map[string]schema) []generatedBodyField {
	fields := map[string]generatedBodyField{}
	collectBodyFields(value, schemas, "", false, fields, map[string]bool{})
	paths := sortedStringKeys(fields)
	result := make([]generatedBodyField, 0, len(paths))
	for _, path := range paths {
		result = append(result, fields[path])
	}
	return result
}

func collectBodyFields(value schema, schemas map[string]schema, prefix string, required bool, fields map[string]generatedBodyField, resolving map[string]bool) {
	if strings.HasPrefix(value.Ref, "#/components/schemas/") {
		if resolving[value.Ref] {
			return
		}
		resolved, ok := schemas[strings.TrimPrefix(value.Ref, "#/components/schemas/")]
		if !ok {
			return
		}
		resolving[value.Ref] = true
		defer delete(resolving, value.Ref)
		collectBodyFields(resolved, schemas, prefix, required, fields, resolving)
		return
	}
	for _, member := range value.AllOf {
		collectBodyFields(member, schemas, prefix, required, fields, resolving)
	}
	for _, member := range value.AnyOf {
		collectBodyFields(member, schemas, prefix, required, fields, resolving)
	}
	for _, member := range value.OneOf {
		collectBodyFields(member, schemas, prefix, required, fields, resolving)
	}
	requiredProperties := map[string]bool{}
	for _, name := range value.Required {
		requiredProperties[name] = true
	}
	for _, name := range sortedKeys(value.Properties) {
		property := value.Properties[name]
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		resolved := resolve(property, schemas)
		if hasNestedFields(resolved) {
			collectBodyFields(property, schemas, path, required || requiredProperties[name], fields, resolving)
			continue
		}
		enum := stringEnum(resolved.Enum)
		if len(enum) == 0 && resolved.Items != nil {
			enum = stringEnum(resolve(*resolved.Items, schemas).Enum)
		}
		fields[path] = generatedBodyField{Path: path, Type: scalarType(resolved), Format: resolved.Format, Description: resolved.Description, Example: resolved.Example, Required: required || requiredProperties[name], Enum: enum}
	}
}

func hasNestedFields(value schema) bool {
	return len(value.Properties) > 0 || len(value.AllOf) > 0 || len(value.AnyOf) > 0 || len(value.OneOf) > 0 || value.Ref != ""
}

// hasRequiredComplexProperty preserves composed request schemas while applying
// the body-only rule to required complex properties from every allOf member.
func hasRequiredComplexProperty(value schema, schemas map[string]schema, resolving map[string]bool) bool {
	if strings.HasPrefix(value.Ref, "#/components/schemas/") {
		if resolving[value.Ref] {
			return false
		}
		resolved, ok := schemas[strings.TrimPrefix(value.Ref, "#/components/schemas/")]
		if !ok {
			return false
		}
		resolving[value.Ref] = true
		defer delete(resolving, value.Ref)
		return hasRequiredComplexProperty(resolved, schemas, resolving)
	}
	for _, name := range value.Required {
		property, ok := value.Properties[name]
		if ok && !isScalar(scalarType(resolve(property, schemas))) {
			return true
		}
	}
	for _, member := range value.AllOf {
		if hasRequiredComplexProperty(member, schemas, resolving) {
			return true
		}
	}
	return false
}

func successMedia(responses map[string]response) string {
	keys := sortedStringKeys(responses)
	for _, status := range keys {
		if !strings.HasPrefix(status, "2") {
			continue
		}
		for _, mediaType := range sortedStringKeys(responses[status].Content) {
			return mediaType
		}
	}
	return ""
}

func validateGeneration(generation string) error {
	if _, ok := coreGenerationRank[generation]; ok {
		return nil
	}
	if _, ok := sideFamilyPrefixes[generation]; ok {
		return nil
	}
	return fmt.Errorf("unsupported API generation %q", generation)
}

func assignCommands(operations []generatedOperation) {
	for index := range operations {
		operation := &operations[index]
		if operation.Generation == "pipelines-v0" && (operation.Noun == "command" || operation.Noun == "operation") {
			operation.Noun = "pipeline-" + operation.Noun
		}
	}
	type candidate struct{ index, rank int }
	groups := map[string][]candidate{}
	for index := range operations {
		operation := &operations[index]
		if operation.AliasOf != "" {
			continue
		}
		rank, _ := coreGenerationRank[operation.Generation]
		key := operation.Noun + "\x00" + operation.Verb
		groups[key] = append(groups[key], candidate{index: index, rank: rank})
	}
	for _, candidates := range groups {
		hasUnscopedCore := false
		for _, candidate := range candidates {
			operation := operations[candidate.index]
			if _, core := coreGenerationRank[operation.Generation]; core && operation.ScopeParent == "" {
				hasUnscopedCore = true
			}
		}
		if !hasUnscopedCore {
			continue
		}
		for _, candidate := range candidates {
			operation := &operations[candidate.index]
			if prefix, side := sideFamilyPrefixes[operation.Generation]; side && operation.Generation != "pipelines-v0" && operation.ScopeParent == "" {
				operation.Noun = prefix + "-" + operation.Noun
			}
		}
	}
	groups = map[string][]candidate{}
	for index := range operations {
		operation := &operations[index]
		if operation.AliasOf != "" {
			continue
		}
		rank, _ := coreGenerationRank[operation.Generation]
		key := operation.Noun + "\x00" + operation.Verb
		groups[key] = append(groups[key], candidate{index: index, rank: rank})
	}
	for _, candidates := range groups {
		maxRank := -1
		for _, candidate := range candidates {
			if candidate.rank > maxRank {
				maxRank = candidate.rank
			}
		}
		for _, candidate := range candidates {
			operation := &operations[candidate.index]
			if _, core := coreGenerationRank[operation.Generation]; core && candidate.rank < maxRank {
				operation.Command = []string{"api", operation.Generation, operation.Noun, operation.Verb}
			} else {
				operation.Command = []string{operation.Noun, operation.Verb}
			}
		}
	}
	byID := map[string]generatedOperation{}
	for _, operation := range operations {
		if operation.OperationID != "" && operation.AliasOf == "" {
			byID[operation.OperationID] = operation
		}
	}
	for index := range operations {
		operation := &operations[index]
		if operation.AliasOf == "" {
			continue
		}
		canonical, ok := byID[operation.AliasOf]
		if !ok {
			continue
		}
		operation.Command = canonical.Command
		operation.Noun = canonical.Noun
		operation.Verb = canonical.Verb
	}
}

func scalarType(value schema) string {
	switch typed := value.Type.(type) {
	case string:
		return typed
	case []any:
		for _, item := range typed {
			if text, ok := item.(string); ok && text != "null" {
				return text
			}
		}
	}
	return "object"
}
func isScalar(kind string) bool {
	return kind == "string" || kind == "number" || kind == "integer" || kind == "boolean"
}
func stringEnum(values []any) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, fmt.Sprint(value))
	}
	return result
}
func sortedKeys(values map[string]schema) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
func sortedStringKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
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
func fatal(err error) { fmt.Fprintln(os.Stderr, "error:", err); os.Exit(1) }
