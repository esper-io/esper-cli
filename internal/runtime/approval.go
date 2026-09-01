package runtime

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const approvalLifetime = 15 * time.Minute

type ApprovalSpec struct {
	BaseURL           string
	Method            string
	Path              string
	ContentType       string
	Query             map[string][]string
	Headers           map[string][]string
	Body              []byte
	AdditionalTargets []ApprovalTarget
}

type ApprovalTarget struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

type ApprovalRequest struct {
	ID                string           `json:"id"`
	Fingerprint       string           `json:"fingerprint"`
	BaseURL           string           `json:"base_url"`
	Method            string           `json:"method"`
	Path              string           `json:"path"`
	ContentType       string           `json:"content_type,omitempty"`
	QueryKeys         []string         `json:"query_keys,omitempty"`
	HeaderKeys        []string         `json:"header_keys,omitempty"`
	BodySHA256        string           `json:"body_sha256,omitempty"`
	BodyFields        []string         `json:"body_fields,omitempty"`
	AdditionalTargets []ApprovalTarget `json:"additional_targets,omitempty"`
	CreatedAt         time.Time        `json:"created_at"`
	ExpiresAt         time.Time        `json:"expires_at"`
	ApprovedAt        time.Time        `json:"approved_at,omitempty"`
	ConsumedAt        time.Time        `json:"consumed_at,omitempty"`
}

type approvalDocument struct {
	Requests []ApprovalRequest `json:"requests"`
}

type ApprovalStore struct {
	Path string
	Now  func() time.Time
}

var (
	approvalOverrideMu sync.Mutex
	approvalOverride   func(ApprovalSpec) error
)

func DefaultApprovalPath() (string, error) {
	credentialsPath, err := DefaultCredentialsPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(credentialsPath), "approvals.json"), nil
}

func NewApprovalStore() (*ApprovalStore, error) {
	path, err := DefaultApprovalPath()
	if err != nil {
		return nil, err
	}
	return &ApprovalStore{Path: path}, nil
}

func (store *ApprovalStore) Request(spec ApprovalSpec) (ApprovalRequest, bool, error) {
	return store.withLock(func(document *approvalDocument) (ApprovalRequest, bool, error) {
		now := store.now()
		document.Requests = retainUnexpired(document.Requests, now)
		fingerprint := approvalFingerprint(spec)
		for _, request := range document.Requests {
			if request.Fingerprint == fingerprint && request.ConsumedAt.IsZero() {
				return request, false, nil
			}
		}
		request, err := newApprovalRequest(spec, fingerprint, now)
		if err != nil {
			return ApprovalRequest{}, false, err
		}
		document.Requests = append(document.Requests, request)
		return request, true, nil
	})
}

func (store *ApprovalStore) Approve(id string) (ApprovalRequest, error) {
	request, _, err := store.withLock(func(document *approvalDocument) (ApprovalRequest, bool, error) {
		now := store.now()
		document.Requests = retainUnexpired(document.Requests, now)
		for index := range document.Requests {
			request := &document.Requests[index]
			if request.ID != id {
				continue
			}
			if !request.ConsumedAt.IsZero() {
				return ApprovalRequest{}, false, fmt.Errorf("approval %s has already been used", id)
			}
			request.ApprovedAt = now
			return *request, true, nil
		}
		return ApprovalRequest{}, false, fmt.Errorf("approval %s was not found or has expired", id)
	})
	return request, err
}

func (store *ApprovalStore) Show(id string) (ApprovalRequest, error) {
	request, _, err := store.withLock(func(document *approvalDocument) (ApprovalRequest, bool, error) {
		now := store.now()
		document.Requests = retainUnexpired(document.Requests, now)
		for _, request := range document.Requests {
			if request.ID == id {
				return request, false, nil
			}
		}
		return ApprovalRequest{}, false, fmt.Errorf("approval %s was not found or has expired", id)
	})
	return request, err
}

func (store *ApprovalStore) Consume(spec ApprovalSpec) (ApprovalRequest, bool, error) {
	return store.withLock(func(document *approvalDocument) (ApprovalRequest, bool, error) {
		now := store.now()
		document.Requests = retainUnexpired(document.Requests, now)
		fingerprint := approvalFingerprint(spec)
		for index := range document.Requests {
			request := &document.Requests[index]
			if request.Fingerprint != fingerprint || request.ApprovedAt.IsZero() || !request.ConsumedAt.IsZero() {
				continue
			}
			request.ConsumedAt = now
			return *request, true, nil
		}
		return ApprovalRequest{}, false, nil
	})
}

func RequireApproval(spec ApprovalSpec) (ApprovalRequest, bool, error) {
	approvalOverrideMu.Lock()
	override := approvalOverride
	approvalOverrideMu.Unlock()
	if override != nil {
		if err := override(spec); err != nil {
			return ApprovalRequest{}, false, err
		}
		return ApprovalRequest{}, true, nil
	}
	store, err := NewApprovalStore()
	if err != nil {
		return ApprovalRequest{}, false, err
	}
	request, approved, err := store.Consume(spec)
	if err != nil || approved {
		return request, approved, err
	}
	request, _, err = store.Request(spec)
	return request, false, err
}

// SetApprovalOverrideForTesting bypasses approval persistence only for in-process tests.
// It is not exposed through the CLI or environment.
func SetApprovalOverrideForTesting(override func(ApprovalSpec) error) func() {
	approvalOverrideMu.Lock()
	previous := approvalOverride
	approvalOverride = override
	approvalOverrideMu.Unlock()
	return func() {
		approvalOverrideMu.Lock()
		approvalOverride = previous
		approvalOverrideMu.Unlock()
	}
}

func RequireTerminal(reader io.Reader) error {
	file, ok := reader.(*os.File)
	if !ok {
		return fmt.Errorf("approval must be run from an interactive terminal")
	}
	info, err := file.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return fmt.Errorf("approval must be run from an interactive terminal")
	}
	return nil
}

func (store *ApprovalStore) withLock(action func(*approvalDocument) (ApprovalRequest, bool, error)) (ApprovalRequest, bool, error) {
	if err := os.MkdirAll(filepath.Dir(store.Path), 0o700); err != nil {
		return ApprovalRequest{}, false, err
	}
	if err := os.Chmod(filepath.Dir(store.Path), 0o700); err != nil {
		return ApprovalRequest{}, false, err
	}
	unlock, err := store.lock()
	if err != nil {
		return ApprovalRequest{}, false, err
	}
	defer unlock()
	document, err := store.load()
	if err != nil {
		return ApprovalRequest{}, false, err
	}
	before, err := json.Marshal(document)
	if err != nil {
		return ApprovalRequest{}, false, err
	}
	request, changed, err := action(&document)
	if err != nil {
		return ApprovalRequest{}, false, err
	}
	after, err := json.Marshal(document)
	if err != nil {
		return ApprovalRequest{}, false, err
	}
	if !bytes.Equal(before, after) {
		if err := store.save(document); err != nil {
			return ApprovalRequest{}, false, err
		}
	}
	return request, changed, nil
}

func (store *ApprovalStore) lock() (func(), error) {
	path := store.Path + ".lock"
	deadline := time.Now().Add(2 * time.Second)
	for {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_ = file.Close()
			return func() { _ = os.Remove(path) }, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		if info, statErr := os.Stat(path); statErr == nil && time.Since(info.ModTime()) > 5*time.Minute {
			_ = os.Remove(path)
			continue
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("approval store is busy")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (store *ApprovalStore) load() (approvalDocument, error) {
	data, err := os.ReadFile(store.Path)
	if errors.Is(err, os.ErrNotExist) {
		return approvalDocument{}, nil
	}
	if err != nil {
		return approvalDocument{}, err
	}
	var document approvalDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return approvalDocument{}, err
	}
	return document, nil
}

func (store *ApprovalStore) save(document approvalDocument) error {
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	temporary, err := os.CreateTemp(filepath.Dir(store.Path), ".approvals-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, store.Path); err != nil {
		return err
	}
	return os.Chmod(store.Path, 0o600)
}

func (store *ApprovalStore) now() time.Time {
	if store.Now != nil {
		return store.Now().UTC()
	}
	return time.Now().UTC()
}

func newApprovalRequest(spec ApprovalSpec, fingerprint string, now time.Time) (ApprovalRequest, error) {
	identifier := make([]byte, 12)
	if _, err := rand.Read(identifier); err != nil {
		return ApprovalRequest{}, err
	}
	request := ApprovalRequest{
		ID:                hex.EncodeToString(identifier),
		Fingerprint:       fingerprint,
		BaseURL:           spec.BaseURL,
		Method:            strings.ToUpper(spec.Method),
		Path:              spec.Path,
		ContentType:       spec.ContentType,
		QueryKeys:         sortedKeys(spec.Query),
		HeaderKeys:        sortedKeys(spec.Headers),
		AdditionalTargets: canonicalTargets(spec.AdditionalTargets),
		CreatedAt:         now,
		ExpiresAt:         now.Add(approvalLifetime),
	}
	if len(spec.Body) > 0 {
		hash := sha256.Sum256(spec.Body)
		request.BodySHA256 = hex.EncodeToString(hash[:])
		request.BodyFields = bodyFields(spec.Body)
	}
	return request, nil
}

func approvalFingerprint(spec ApprovalSpec) string {
	payload := struct {
		BaseURL, Method, Path, ContentType string
		Query, Headers                     map[string][]string
		BodySHA256                         string
		AdditionalTargets                  []ApprovalTarget
	}{
		BaseURL: strings.TrimRight(spec.BaseURL, "/"), Method: strings.ToUpper(spec.Method), Path: spec.Path, ContentType: spec.ContentType,
		Query: canonicalValues(spec.Query), Headers: canonicalValues(spec.Headers), AdditionalTargets: canonicalTargets(spec.AdditionalTargets),
	}
	if len(spec.Body) > 0 {
		hash := sha256.Sum256(spec.Body)
		payload.BodySHA256 = hex.EncodeToString(hash[:])
	}
	data, _ := json.Marshal(payload)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func canonicalValues(values map[string][]string) map[string][]string {
	result := make(map[string][]string, len(values))
	for key, value := range values {
		copyValue := append([]string(nil), value...)
		sort.Strings(copyValue)
		result[key] = copyValue
	}
	return result
}

func canonicalTargets(targets []ApprovalTarget) []ApprovalTarget {
	result := append([]ApprovalTarget(nil), targets...)
	for index := range result {
		result[index].Method = strings.ToUpper(result[index].Method)
	}
	sort.Slice(result, func(left, right int) bool {
		if result[left].Method != result[right].Method {
			return result[left].Method < result[right].Method
		}
		return result[left].Path < result[right].Path
	})
	return result
}

func bodyFields(data []byte) []string {
	var value any
	if json.Unmarshal(data, &value) != nil {
		return nil
	}
	fields := map[string]bool{}
	collectBodyFields(value, "", fields)
	result := make([]string, 0, len(fields))
	for field := range fields {
		result = append(result, field)
	}
	sort.Strings(result)
	return result
}

func collectBodyFields(value any, prefix string, fields map[string]bool) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			path := key
			if prefix != "" {
				path = prefix + "." + key
			}
			fields[path] = true
			collectBodyFields(child, path, fields)
		}
	case []any:
		for _, child := range typed {
			collectBodyFields(child, prefix, fields)
		}
	}
}

func retainUnexpired(requests []ApprovalRequest, now time.Time) []ApprovalRequest {
	result := make([]ApprovalRequest, 0, len(requests))
	for _, request := range requests {
		if request.ExpiresAt.After(now) {
			result = append(result, request)
		}
	}
	return result
}

func sortedKeys(values map[string][]string) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}
