package errornotifier

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/theplant/appkit/log"
)

type key int

const ctxKey key = iota

// Recover wraps an http.Handler to report all `panic`s to Airbrake.
func Recover(n Notifier) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			c := context.WithValue(req.Context(), ctxKey, n)
			err := NotifyOnPanic(n, req, func() {
				h.ServeHTTP(w, req.WithContext(c))
			})
			if err != nil {
				panic(err)
			}
		})
	}
}

// ForceContext extracts a notifier from the request context, falling
// back to a LogNotifier using the context's logger.
func ForceContext(c context.Context) Notifier {
	if c != nil {
		notifier, ok := c.Value(ctxKey).(Notifier)
		if ok {
			return notifier
		}
	}

	return NewLogNotifier(log.ForceContext(c))
}

// NotifyOnPanic will notify Airbrake if function f panics, and will
// return the error that caused the panic (if any)
//
// This is for wrapping Goroutines to prevent panics from bringing
// down the whole application.
func NotifyOnPanic(n Notifier, req *http.Request, f func()) (err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}

		if e, ok := r.(error); !ok {
			err = fmt.Errorf("%v", r)
		} else {
			err = e
		}

		l := log.ForceContext(req.Context())
		l.Error().Log("msg", err.Error(), "stack", string(debug.Stack()))

		// not using goroutine here in order to keep the whole backtrace in
		// airbrake report
		n.Notify(err, req)
		return
	}()

	f()
	return
}
