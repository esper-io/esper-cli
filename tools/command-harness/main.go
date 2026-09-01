package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/esper-io/esper-cli/internal/cmd"
	"github.com/esper-io/esper-cli/internal/cmd/generated"
)

const mutationConfirmation = "I_UNDERSTAND_THIS_TENANT_IS_DISPOSABLE"
const deviceMutationConfirmation = "I_UNDERSTAND_THIS_DEVICE_IS_DISPOSABLE"

type result struct {
	Name       string `json:"name"`
	Command    string `json:"command"`
	Method     string `json:"method,omitempty"`
	Path       string `json:"path,omitempty"`
	Status     string `json:"status"`
	ExitCode   int    `json:"exit_code,omitempty"`
	Detail     string `json:"detail,omitempty"`
	DurationMS int64  `json:"duration_ms,omitempty"`
}

type report struct {
	Mode      string   `json:"mode"`
	StartedAt string   `json:"started_at"`
	Finished  string   `json:"finished_at"`
	Counts    counts   `json:"counts"`
	Results   []result `json:"results"`
}

type counts struct {
	Pass    int `json:"pass"`
	Fail    int `json:"fail"`
	Skipped int `json:"skipped"`
	Alias   int `json:"alias"`
}

type scenario struct {
	Name                 string `json:"name"`
	DisposableEnterprise string `json:"disposable_enterprise"`
	DisposableScope      string `json:"disposable_scope"`
	DisposableDevice     string `json:"disposable_device"`
	Steps                []step `json:"steps"`
	Cleanup              []step `json:"cleanup"`
}

type step struct {
	Name         string   `json:"name"`
	Args         []string `json:"args"`
	ExpectedExit *int     `json:"expected_exit"`
	Mutation     bool     `json:"mutation"`
}

type harness struct {
	binary                string
	enterprise            string
	device                string
	group                 string
	app                   string
	timeout               time.Duration
	known                 map[string]string
	allowDefaultPageLists bool
}

func main() {
	mode := flag.String("mode", "offline", "offline, live-readonly, or live-mutations")
	binary := flag.String("binary", "", "espercli binary to execute for live modes")
	enterprise := flag.String("enterprise", "", "disposable/live enterprise ID")
	device := flag.String("device", "", "owned/live device ID")
	group := flag.String("group", "", "safe group ID for dependent reads")
	app := flag.String("app", "", "safe application ID for dependent reads")
	scenarioPath := flag.String("scenario", "", "mutation scenario JSON path")
	reportPath := flag.String("report", "dist/command-harness-report.json", "JSON report path")
	allowMutations := flag.Bool("allow-mutations", false, "allow mutation scenario execution")
	allowDefaultPageLists := flag.Bool("allow-default-page-lists", false, "allow GET list operations without a limit parameter")
	confirmation := flag.String("confirmation", "", "must equal the disposable-tenant confirmation phrase")
	disposableDevice := flag.Bool("disposable-device", false, "allow only device-scoped mutations for the disposable --device")
	flag.Parse()

	started := time.Now().UTC()
	var results []result
	switch *mode {
	case "offline":
		results = runOffline()
	case "live-readonly":
		if *binary == "" || *enterprise == "" || *device == "" {
			fatal("live-readonly requires --binary, --enterprise, and --device")
		}
		results = (&harness{binary: *binary, enterprise: *enterprise, device: *device, group: *group, app: *app, timeout: 45 * time.Second, allowDefaultPageLists: *allowDefaultPageLists}).runReadonly()
	case "live-mutations":
		if *binary == "" || *enterprise == "" || *device == "" || *scenarioPath == "" {
			fatal("live-mutations requires --binary, --enterprise, --device, and --scenario")
		}
		expectedConfirmation := mutationConfirmation
		if *disposableDevice {
			expectedConfirmation = deviceMutationConfirmation
		}
		if !*allowMutations || *confirmation != expectedConfirmation {
			fatal("live-mutations requires --allow-mutations and the exact disposable confirmation")
		}
		results = runMutations(*binary, *enterprise, *device, *scenarioPath, *confirmation, *disposableDevice)
	default:
		fatal(fmt.Sprintf("unknown mode %q", *mode))
	}

	finished := time.Now().UTC()
	currentReport := report{Mode: *mode, StartedAt: started.Format(time.RFC3339), Finished: finished.Format(time.RFC3339), Results: results}
	for _, item := range results {
		switch item.Status {
		case "PASS":
			currentReport.Counts.Pass++
		case "FAIL":
			currentReport.Counts.Fail++
		case "SKIPPED":
			currentReport.Counts.Skipped++
		case "ALIAS":
			currentReport.Counts.Alias++
		}
	}
	if err := writeReport(*reportPath, currentReport); err != nil {
		fatal(err.Error())
	}
	for _, item := range results {
		fmt.Printf("%s\t%s\t%s\n", item.Status, item.Command, item.Detail)
	}
	fmt.Printf("TOTAL\tPASS=%d\tFAIL=%d\tSKIPPED=%d\tALIAS=%d\n", currentReport.Counts.Pass, currentReport.Counts.Fail, currentReport.Counts.Skipped, currentReport.Counts.Alias)
	if currentReport.Counts.Fail > 0 {
		os.Exit(1)
	}
}

func runOffline() []result {
	operations := generated.Operations()
	results := make([]result, 0, len(operations)+8)
	seen := map[string]bool{}
	for _, operation := range operations {
		commandPath := strings.Join(operation.Command, " ")
		item := result{Name: operation.OperationID, Command: commandPath, Method: operation.Method, Path: operation.Path}
		if operation.AliasOf != "" {
			item.Status = "ALIAS"
			item.Detail = "covered by " + operation.AliasOf
			results = append(results, item)
			continue
		}
		if seen[commandPath] {
			item.Status = "PASS"
			item.Detail = "shared command help already checked"
			results = append(results, item)
			continue
		}
		seen[commandPath] = true
		root := cmd.NewRootCommand()
		command, _, err := root.Find(operation.Command)
		if err != nil || command == root {
			item.Status = "FAIL"
			item.Detail = "command is not reachable"
			results = append(results, item)
			continue
		}
		var output bytes.Buffer
		root.SetOut(&output)
		root.SetErr(&output)
		root.SetArgs(append(append([]string{}, operation.Command...), "--help"))
		if err := root.Execute(); err != nil {
			item.Status = "FAIL"
			item.Detail = err.Error()
		} else {
			item.Status = "PASS"
			item.Detail = "help rendered"
		}
		results = append(results, item)
	}
	for _, commandPath := range [][]string{{"configure"}, {"configure", "show"}, {"context", "set"}, {"context", "get"}, {"context", "clear"}, {"discover"}, {"secureadb", "connect"}, {"completion", "bash"}, {"completion", "zsh"}, {"completion", "fish"}, {"completion", "powershell"}, {"version"}} {
		root := cmd.NewRootCommand()
		item := result{Name: "handwritten", Command: strings.Join(commandPath, " ")}
		if commandPath[0] != "completion" {
			if command, _, err := root.Find(commandPath); err != nil || command == root {
				item.Status = "FAIL"
				item.Detail = "command is not reachable"
				results = append(results, item)
				continue
			}
		}
		var output bytes.Buffer
		root.SetOut(&output)
		root.SetErr(&output)
		root.SetArgs(append(append([]string{}, commandPath...), "--help"))
		if err := root.Execute(); err != nil {
			item.Status = "FAIL"
			item.Detail = err.Error()
		} else {
			item.Status = "PASS"
			item.Detail = "help rendered"
		}
		results = append(results, item)
	}
	return results
}

func (runner *harness) runReadonly() []result {
	known := map[string]string{
		"enterprise": runner.enterprise, "enterprise_id": runner.enterprise,
		"device": runner.device, "device_id": runner.device, "deviceId": runner.device,
		"group": runner.group, "app": runner.app, "application": runner.app,
	}
	var pending []generated.Operation
	for _, operation := range generated.Operations() {
		if operation.Method == "GET" && operation.AliasOf == "" {
			pending = append(pending, operation)
		}
	}
	var results []result
	for pass := 0; pass < 4 && len(pending) > 0; pass++ {
		var deferred []generated.Operation
		progress := false
		for _, operation := range pending {
			if runner.skipsUnboundedList(operation) {
				results = append(results, result{Name: operation.OperationID, Command: strings.Join(operation.Command, " "), Method: operation.Method, Path: operation.Path, Status: "SKIPPED", Detail: "list has no bounded limit parameter"})
				progress = true
				continue
			}
			args, missing := readonlyArgs(operation, known)
			if len(missing) > 0 {
				deferred = append(deferred, operation)
				continue
			}
			item, output := runner.execute(operation, args)
			results = append(results, item)
			if item.Status == "PASS" {
				collectKnown(output, operation.Noun, known)
			}
			progress = true
		}
		pending = deferred
		if !progress {
			break
		}
	}
	for _, operation := range pending {
		_, missing := readonlyArgs(operation, known)
		results = append(results, result{Name: operation.OperationID, Command: strings.Join(operation.Command, " "), Method: operation.Method, Path: operation.Path, Status: "SKIPPED", Detail: "missing safe input: " + strings.Join(missing, ", ")})
	}
	return results
}

func (runner *harness) skipsUnboundedList(operation generated.Operation) bool {
	return operation.Verb == "list" && !hasParameter(operation, "limit") && !runner.allowDefaultPageLists
}

func runMutations(binary, enterprise, device, path, confirmation string, disposableDevice bool) []result {
	data, err := os.ReadFile(path)
	if err != nil {
		fatal(fmt.Sprintf("read mutation scenario: %v", err))
	}
	var plan scenario
	if err := json.Unmarshal(data, &plan); err != nil {
		fatal(fmt.Sprintf("decode mutation scenario: %v", err))
	}
	variables := map[string]string{"ENTERPRISE": enterprise, "DEVICE": device}
	if disposableDevice {
		if plan.DisposableScope != "device" || expand(plan.DisposableDevice, variables) != device {
			fatal("device-scoped scenario must identify the exact --device as disposable")
		}
	} else if plan.DisposableEnterprise != "${ENTERPRISE}" && expand(plan.DisposableEnterprise, variables) != enterprise {
		fatal("scenario disposable_enterprise does not match --enterprise")
	}
	expectedConfirmation := mutationConfirmation
	if disposableDevice {
		expectedConfirmation = deviceMutationConfirmation
	}
	if confirmation != expectedConfirmation || len(plan.Cleanup) == 0 {
		fatal("mutation scenario requires cleanup and disposable confirmation")
	}
	runner := &harness{binary: binary, timeout: 60 * time.Second}
	results := make([]result, 0, len(plan.Steps)+len(plan.Cleanup))
	failed := false
	for _, item := range plan.Steps {
		result := runner.executeStep(item, variables, disposableDevice)
		results = append(results, result)
		if result.Status == "FAIL" {
			failed = true
			break
		}
	}
	for index := len(plan.Cleanup) - 1; index >= 0; index-- {
		results = append(results, runner.executeStep(plan.Cleanup[index], variables, disposableDevice))
	}
	if failed {
		for index := range results {
			if results[index].Status == "PASS" && results[index].Name == "" {
				results[index].Status = "FAIL"
			}
		}
	}
	return results
}

func (runner *harness) executeStep(item step, variables map[string]string, disposableDevice bool) result {
	if item.ExpectedExit == nil {
		return result{Name: item.Name, Status: "FAIL", Detail: "scenario step must declare expected_exit"}
	}
	args := make([]string, len(item.Args))
	for index, arg := range item.Args {
		args[index] = expand(arg, variables)
	}
	operations := matchingOperations(args)
	if item.Mutation && disposableDevice && !contains(args, variables["DEVICE"]) {
		return result{Name: item.Name, Command: safeCommand(args), Status: "FAIL", Detail: "device-scoped mutation must target the disposable device"}
	}
	for _, operation := range operations {
		if operation.Method != "GET" && !item.Mutation {
			return result{Name: item.Name, Command: safeCommand(args), Status: "FAIL", Detail: "non-GET scenario step must set mutation=true"}
		}
		if operation.Destructive && !contains(args, "--yes") {
			return result{Name: item.Name, Command: safeCommand(args), Status: "FAIL", Detail: "destructive mutation must explicitly include --yes"}
		}
	}
	if item.Mutation && len(operations) == 0 && !contains(args, "--yes") {
		return result{Name: item.Name, Command: safeCommand(args), Status: "FAIL", Detail: "unrecognized mutation must explicitly include --yes"}
	}
	result, _ := runner.executeArgs(item.Name, args)
	if result.ExitCode == *item.ExpectedExit && result.Detail != "process start failed" && result.Detail != "timeout" {
		result.Status = "PASS"
		result.Detail = fmt.Sprintf("exit %d", result.ExitCode)
	} else {
		result.Status = "FAIL"
		result.Detail = fmt.Sprintf("exit %d, expected %d", result.ExitCode, *item.ExpectedExit)
	}
	return result
}

func (runner *harness) execute(operation generated.Operation, args []string) (result, []byte) {
	item, output := runner.executeArgs(strings.Join(operation.Command, " "), args)
	item.Method = operation.Method
	item.Path = operation.Path
	return item, output
}

func (runner *harness) executeArgs(name string, args []string) (result, []byte) {
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), runner.timeout)
	defer cancel()
	command := exec.CommandContext(ctx, runner.binary, args...)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	item := result{Name: name, Command: safeCommand(args), Status: "PASS", ExitCode: 0, DurationMS: time.Since(started).Milliseconds()}
	if err != nil {
		item.Status = "FAIL"
		item.ExitCode = processExitCode(err)
		if ctx.Err() != nil {
			item.ExitCode = -1
			item.Detail = "timeout"
		} else if _, ok := err.(*exec.ExitError); !ok {
			item.ExitCode = -1
			item.Detail = "process start failed"
		} else {
			item.Detail = failedCommandDetail(stderr.String())
		}
	}
	return item, stdout.Bytes()
}

func failedCommandDetail(stderr string) string {
	value := strings.TrimSpace(stderr)
	if value == "" {
		return "command failed"
	}
	const maxLength = 512
	if len(value) > maxLength {
		value = value[:maxLength] + "..."
	}
	return value
}

func matchingOperations(args []string) []generated.Operation {
	var matches []generated.Operation
	for _, operation := range generated.Operations() {
		if operation.AliasOf != "" || len(args) < len(operation.Command) {
			continue
		}
		match := true
		for index, segment := range operation.Command {
			if args[index] != segment {
				match = false
				break
			}
		}
		if match {
			matches = append(matches, operation)
		}
	}
	return matches
}

func readonlyArgs(operation generated.Operation, known map[string]string) ([]string, []string) {
	args := append([]string{}, operation.Command...)
	var missing []string
	for _, parameter := range pathParameters(operation) {
		value := valueFor(parameter, operation, known)
		if value == "" {
			missing = append(missing, parameter.Name)
			continue
		}
		if parameter.Scope {
			name := parameter.ScopeName
			if name == "" {
				name = inferResource(parameter.Name, operation.Noun)
			}
			args = append(args, "--"+kebab(name), value)
		} else {
			args = append(args, value)
		}
	}
	set := map[string]bool{}
	for _, parameter := range operation.Parameters {
		if parameter.In != "query" && parameter.In != "header" {
			continue
		}
		if parameter.Name == "limit" {
			args = append(args, "--limit", "1")
			continue
		}
		if !parameter.Required {
			continue
		}
		value := valueFor(parameter, operation, known)
		if value == "" {
			missing = append(missing, parameter.Name)
			continue
		}
		args = append(args, "--"+kebab(parameter.Name), value)
		set[parameter.Name] = true
	}
	if len(operation.RequireOneOf) > 0 {
		selected := false
		for _, name := range operation.RequireOneOf {
			if set[name] {
				selected = true
				break
			}
		}
		if !selected {
			for _, name := range operation.RequireOneOf {
				for _, parameter := range operation.Parameters {
					if parameter.Name == name {
						if value := valueFor(parameter, operation, known); value != "" {
							args = append(args, "--"+kebab(name), value)
							selected = true
						}
					}
				}
				if selected {
					break
				}
			}
		}
		if !selected {
			missing = append(missing, strings.Join(operation.RequireOneOf, "|"))
		}
	}
	args = append(args, "--json")
	return args, unique(missing)
}

func hasParameter(operation generated.Operation, name string) bool {
	for _, parameter := range operation.Parameters {
		if parameter.Name == name {
			return true
		}
	}
	return false
}

func pathParameters(operation generated.Operation) []generated.Parameter {
	parameters := make([]generated.Parameter, 0)
	for _, parameter := range operation.Parameters {
		if parameter.In == "path" {
			parameters = append(parameters, parameter)
		}
	}
	sort.SliceStable(parameters, func(left, right int) bool {
		return strings.Index(operation.Path, "{"+parameters[left].Name+"}") < strings.Index(operation.Path, "{"+parameters[right].Name+"}")
	})
	return parameters
}

func valueFor(parameter generated.Parameter, operation generated.Operation, known map[string]string) string {
	if resource := requiredResource(operation, parameter.Name); resource != "" {
		return known[resource]
	}
	if known[parameter.Name] != "" {
		return known[parameter.Name]
	}
	resource := parameter.ScopeName
	if resource == "" {
		resource = inferResource(parameter.Name, operation.Noun)
	}
	if known[resource] != "" {
		return known[resource]
	}
	if len(parameter.Enum) > 0 {
		return parameter.Enum[0]
	}
	switch parameter.Name {
	case "parent_group_ids":
		return known["group"]
	case "start_date":
		return reportDate(-2)
	case "end_date":
		return reportDate(-1)
	case "from_time", "last_seen_gt":
		return time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	case "to_time", "last_seen_lt":
		return time.Now().UTC().Format(time.RFC3339)
	case "period":
		return "day"
	case "statistic":
		return "avg"
	case "category":
		return "device"
	}
	return ""
}

func requiredResource(operation generated.Operation, parameter string) string {
	switch operation.Noun {
	case "app-vpp":
		if parameter == "appId" || parameter == "app_id" {
			return "app-vpp"
		}
	case "device-app":
		if parameter == "app_id" {
			return "device-app"
		}
	case "esper-app-version":
		if parameter == "appId" || parameter == "app_id" {
			return "esper-app"
		}
		if parameter == "versionId" || parameter == "version_id" {
			return "esper-app-version"
		}
	case "tenant-app", "tenant-app-version":
		if parameter == "appId" || parameter == "app_id" {
			return "tenant-app"
		}
		if parameter == "versionId" || parameter == "version_id" {
			return "tenant-app-version"
		}
	case "blueprint", "blueprint-revision", "revision":
		if operation.Generation == "legacy" && parameter == "blueprint_id" {
			return "legacy-blueprint"
		}
	case "blueprint-version":
		if parameter == "blueprint_id" {
			return "blueprint"
		}
		if parameter == "version_id" {
			return "blueprint-version"
		}
	}
	return ""
}

func reportDate(daysAgo int) string {
	return time.Now().UTC().AddDate(0, 0, daysAgo).Truncate(24 * time.Hour).Format("2006-01-02T15:04:05")
}

func collectKnown(data []byte, noun string, known map[string]string) {
	var value any
	if json.Unmarshal(data, &value) == nil {
		collectValue(value, noun, known)
	}
}

func collectValue(value any, noun string, known map[string]string) {
	switch typed := value.(type) {
	case []any:
		for _, child := range typed {
			collectValue(child, noun, known)
		}
	case map[string]any:
		if id, ok := typed["id"].(string); ok && known[noun] == "" {
			known[noun] = id
		}
		for key, child := range typed {
			if id, ok := child.(string); ok && (strings.HasSuffix(key, "_id") || strings.HasSuffix(key, "Id")) {
				resource := inferResource(key, noun)
				if resource == "user" && noun != "user" {
					collectValue(child, noun, known)
					continue
				}
				if noun == "blueprint" && key == "latest_published_version_id" {
					resource = "blueprint-version"
				}
				if known[resource] == "" {
					known[resource] = id
				}
			}
			collectValue(child, noun, known)
		}
	}
}

func inferResource(parameter, noun string) string {
	if parameter == "id" {
		return noun
	}
	value := strings.TrimSuffix(kebab(parameter), "-id")
	switch value {
	case "operations":
		return "operation"
	case "devicegroup":
		return "device-group"
	case "application", "app":
		return "application"
	default:
		return value
	}
}

func kebab(value string) string {
	var output []rune
	for index, character := range value {
		if character == '_' {
			output = append(output, '-')
		} else {
			if index > 0 && character >= 'A' && character <= 'Z' {
				output = append(output, '-')
			}
			output = append(output, character)
		}
	}
	return strings.ToLower(string(output))
}

func unique(values []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func expand(value string, variables map[string]string) string {
	for key, replacement := range variables {
		value = strings.ReplaceAll(value, "${"+key+"}", replacement)
	}
	return value
}

func safeCommand(args []string) string {
	redactNext := false
	var safe []string
	for _, arg := range args {
		if redactNext {
			safe = append(safe, "<redacted>")
			redactNext = false
			continue
		}
		lower := strings.ToLower(arg)
		if strings.HasPrefix(lower, "--api-key=") || strings.HasPrefix(lower, "--body=") || strings.HasPrefix(lower, "--authorization=") || strings.HasPrefix(lower, "--token=") || strings.HasPrefix(lower, "--password=") {
			name := arg[:strings.IndexByte(arg, '=')]
			safe = append(safe, name+"=<redacted>")
			continue
		}
		safe = append(safe, arg)
		if arg == "--api-key" || arg == "--body" || arg == "--authorization" || arg == "--token" || arg == "--password" {
			redactNext = true
		}
	}
	return strings.Join(safe, " ")
}

func processExitCode(err error) int {
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return exitError.ExitCode()
	}
	return 1
}

func writeReport(path string, value report) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, "error:", message)
	os.Exit(2)
}
