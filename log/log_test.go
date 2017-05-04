package log_test

import (
	"errors"
	"fmt"
	"go/build"
	"io"
	"strings"
	"testing"

	"bytes"

	klog "github.com/go-kit/kit/log"

	"github.com/theplant/appkit/kerrs"
	"github.com/theplant/appkit/log"
	"github.com/theplant/testingutils"
)

func TestLog(t *testing.T) {
	l := log.Default()
	err := l.Crit().Log("msg", "hello")
	if err != nil {
		t.Error(err)
	}
}

var logErrCases = []struct {
	err      error
	expected string
}{
	{
		err: kerrs.Wrapv(io.EOF, "wrong io", "testcase", "TestLogError", "lineno", 23),
		expected: `
level error testcase TestLogError lineno 23 msg wrong io: EOF stacktrace wrong io testcase=TestLogError lineno=23: EOF
github.com/theplant/appkit/kerrs.Wrapv
	github.com/theplant/appkit/kerrs/errors.go:27
github.com/theplant/appkit/log_test.init
	github.com/theplant/appkit/log/log_test.go:33
main.init
	github.com/theplant/appkit/log/_test/_testmain.go:50
runtime.main
	runtime/proc.go:173
runtime.goexit
	runtime/asm_amd64.s:2197
`,
	},
	{
		err: errors.New("it's error"),
		expected: `
level error msg it's error
`,
	},
	{
		err: kerrs.Wrapv(io.EOF, "the message", "testcase", "TestLogError", "lineno"),
		expected: `
level error testcase TestLogError lineno <value-missing> msg the message: EOF stacktrace the message testcase=TestLogError lineno="<value-missing>": EOF
github.com/theplant/appkit/kerrs.Wrapv
	github.com/theplant/appkit/kerrs/errors.go:27
github.com/theplant/appkit/log_test.init
	github.com/theplant/appkit/log/log_test.go:55
main.init
	github.com/theplant/appkit/log/_test/_testmain.go:50
runtime.main
	runtime/proc.go:173
runtime.goexit
	runtime/asm_amd64.s:2197
`,
	},
}

func TestLogError(t *testing.T) {

	for _, errc := range logErrCases {
		output := bytes.NewBuffer(nil)
		output.WriteString("\n")
		l := log.Default()
		lev := klog.LoggerFunc(func(keyvals ...interface{}) (err error) {
			fmt.Fprintln(output, keyvals...)
			return nil
		})
		l = log.Logger{lev}

		l.WithError(errc.err).Log()

		diff := testingutils.PrettyJsonDiff(errc.expected, cleanStacktrace(output.String()))
		if len(diff) > 0 {
			t.Error(diff)
		}
	}
}

func cleanStacktrace(stacktrace string) (cleantrace string) {
	cleantrace = strings.Replace(stacktrace, build.Default.GOPATH+"/src/", "", -1)
	cleantrace = strings.Replace(cleantrace, build.Default.GOROOT+"/src/", "", -1)
	return
}
