package runtime

import "time"

type ApprovalWindow struct {
	StartedAt time.Time `json:"started_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type ApprovalWindowStatus struct {
	Active bool            `json:"active"`
	Window *ApprovalWindow `json:"window,omitempty"`
}

func windowStatus(window *ApprovalWindow, now time.Time) ApprovalWindowStatus {
	active := window != nil && !window.StartedAt.IsZero() &&
		window.ExpiresAt.Equal(window.StartedAt.Add(approvalLifetime)) &&
		!window.StartedAt.After(now) && window.ExpiresAt.After(now)
	return ApprovalWindowStatus{Active: active, Window: window}
}

func (store *ApprovalStore) WindowStatus() (ApprovalWindowStatus, error) {
	var status ApprovalWindowStatus
	_, _, err := store.withLock(func(document *approvalDocument) (ApprovalRequest, bool, error) {
		status = windowStatus(document.Window, store.now())
		return ApprovalRequest{}, false, nil
	})
	return status, err
}

// StartWindow is called only after terminal confirmation, or by isolated tests.
// Rechecking under the store lock prevents concurrent starts from extending it.
func (store *ApprovalStore) StartWindow() (ApprovalWindowStatus, error) {
	var status ApprovalWindowStatus
	_, _, err := store.withLock(func(document *approvalDocument) (ApprovalRequest, bool, error) {
		now := store.now()
		status = windowStatus(document.Window, now)
		if !status.Active {
			document.Window = &ApprovalWindow{StartedAt: now, ExpiresAt: now.Add(approvalLifetime)}
			status = windowStatus(document.Window, now)
		}
		return ApprovalRequest{}, false, nil
	})
	return status, err
}

func (store *ApprovalStore) StopWindow() (ApprovalWindowStatus, error) {
	_, _, err := store.withLock(func(document *approvalDocument) (ApprovalRequest, bool, error) {
		document.Window = nil
		return ApprovalRequest{}, false, nil
	})
	return ApprovalWindowStatus{}, err
}
