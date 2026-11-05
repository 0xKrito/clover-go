package ui

import (
	"sync/atomic"

	"github.com/egoist/mygo"
)

// Task tracks a single asynchronous operation.
type Task struct {
	Running bool
	Err     error
}

var mainHop atomic.Pointer[func(func())]

// RunOnMain dispatches a function onto the UI main thread.
func RunOnMain(fn func()) {
	if hop := mainHop.Load(); hop != nil {
		(*hop)(fn)
		return
	}
	mygo.RunOnMain(fn)
}

// SetRunOnMain injects a test stub for headless execution.
func SetRunOnMain(hop func(func())) (restore func()) {
	old := mainHop.Swap(&hop)
	return func() { mainHop.Store(old) }
}

// RunTask executes fn on a background goroutine and posts results back to UI thread.
func RunTask(t *Task, post func(func()), fn func() error, done func(error)) {
	if post == nil {
		post = RunOnMain
	}
	if t.Running {
		return
	}
	t.Running, t.Err = true, nil
	go func() {
		err := fn()
		post(func() {
			t.Running, t.Err = false, err
			if done != nil {
				done(err)
			}
		})
	}()
}
