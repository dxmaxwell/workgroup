package workgroup

import (
	"context"
	"sync"
)

// AccumulateManager is used only for testing (for now)
type AccumulateManager struct {
	mutex   sync.Mutex
	manager Manager
	Errors  []error
}

func (m *AccumulateManager) Error() error {
	return m.manager.Error()
}

func (m *AccumulateManager) Manage(ctx context.Context, c Canceller, idx int, err *error) int {
	n := m.manager.Manage(ctx, c, idx, err)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	for len(m.Errors) < n {
		m.Errors = append(m.Errors, nil)
	}
	m.Errors[n-1] = *err

	return n
}
