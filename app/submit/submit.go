package submit

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/xhd2015/todo/log"
)

// SubmitState manages the state of content submission operations
type SubmitState struct {
	mutex          sync.RWMutex
	isSubmitting   bool
	pendingContent string
	onRestore      func(content string) // Callback to restore content on failure
}

// NewSubmitState creates a new SubmitState with the given restore callback
func NewSubmitState(onRestore func(content string)) *SubmitState {
	return &SubmitState{
		onRestore: onRestore,
	}
}

func (s *SubmitState) SetOnRestore(onRestore func(content string)) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.onRestore = onRestore
}

// SetSubmitting sets the submission state to true and stores the pending content
func (s *SubmitState) SetSubmitting(content string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.isSubmitting = true
	s.pendingContent = content
}

// IsSubmitting returns whether a submission is currently in progress
func (s *SubmitState) IsSubmitting() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.isSubmitting
}

// Clear clears the submission state without restoring content
func (s *SubmitState) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.isSubmitting = false
	s.pendingContent = ""
}

// RestorePendingContent restores failed content using the callback and clears submission state
func (s *SubmitState) RestorePendingContent() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.pendingContent != "" && s.onRestore != nil {
		s.onRestore(s.pendingContent)
	}
	s.isSubmitting = false
	s.pendingContent = ""
}

// Do executes a function with proper submission state management
// It handles the common pattern: SetSubmitting → defer Clear() → error handling with restore
// Also handles panics by restoring content and re-panicking
func (s *SubmitState) Do(ctx context.Context, content string, fn func() error) (err error) {
	s.SetSubmitting(content)
	defer s.Clear()

	log.Infof(ctx, "submitting %s", content)

	// Handle panics by restoring content and re-panicking
	defer func() {
		if e := recover(); e != nil {
			// Log panic with stack trace
			log.Errorf(ctx, "submission panic: %v\nstack trace:\n%s", e, debug.Stack())
			if pe, ok := e.(error); ok {
				err = pe
			} else {
				err = fmt.Errorf("panic: %v", e)
			}
		}
		if err != nil {
			// Log submission error
			log.Errorf(ctx, "submission error: %v", err)
			s.RestorePendingContent()
		}
	}()

	err = fn()
	return
}
