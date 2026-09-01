package runtime

import (
	"encoding/json"
	"fmt"
)

type Page struct {
	Count    int
	Next     string
	Previous string
	Results  []json.RawMessage
}

func UnwrapLimitOffset(data []byte) (Page, error) {
	var envelope struct {
		Count    int               `json:"count"`
		Next     *string           `json:"next"`
		Previous *string           `json:"previous"`
		Results  []json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return Page{}, fmt.Errorf("decode limit-offset page: %w", err)
	}
	page := Page{Count: envelope.Count, Results: envelope.Results}
	if envelope.Next != nil {
		page.Next = *envelope.Next
	}
	if envelope.Previous != nil {
		page.Previous = *envelope.Previous
	}
	return page, nil
}

func UnwrapAppsEnvelope(data []byte) (Page, error) {
	var envelope struct {
		Content struct {
			Count    int               `json:"count"`
			Next     *string           `json:"next"`
			Previous *string           `json:"previous"`
			Prev     *string           `json:"prev"`
			Results  []json.RawMessage `json:"results"`
		} `json:"content"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return Page{}, fmt.Errorf("decode apps page: %w", err)
	}
	page := Page{Count: envelope.Content.Count, Results: envelope.Content.Results}
	if envelope.Content.Next != nil {
		page.Next = *envelope.Content.Next
	}
	previous := envelope.Content.Previous
	if previous == nil {
		previous = envelope.Content.Prev
	}
	if previous != nil {
		page.Previous = *previous
	}
	return page, nil
}

func MarshalMergedResults(results []json.RawMessage) ([]byte, error) {
	data, err := json.Marshal(results)
	if err != nil {
		return nil, fmt.Errorf("encode merged results: %w", err)
	}
	return data, nil
}
