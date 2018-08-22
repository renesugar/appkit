package kerrs_test

import (
	"errors"
	"fmt"
	"strings"

	"bytes"

	"github.com/theplant/appkit/kerrs"
	"github.com/theplant/testingutils"
)

func ExampleWrapv_errors() {
	err0 := errors.New("hi, I am an error")
	err1 := kerrs.Wrapv(err0, "wrong", "code", "12123", "value", 12312)

	// fmt.Printf("%+v", err)
	err2 := kerrs.Wrapv(err1, "more explain about the error", "morecontext", "999")

	actual := cleanStacktrace(fmt.Sprintf("%+v\n", err2))
	expected := `more explain about the error morecontext=999: wrong code=12123 value=12312: hi, I am an error
<stacktrace>
`

	diff := testingutils.PrettyJsonDiff(expected, actual)
	fmt.Println(diff)
	// Output:
	//

}

func ExampleAppend_errors() {

	var handleCSV = func(csvContent string) (err error) {
		var handleLine = func(line string) (err error) {
			if len(line) > 3 {
				err = fmt.Errorf("Invalid Length for %s", line)
			}
			return
		}
		lines := strings.Split(csvContent, "\n")
		for _, line := range lines {
			lineErr := handleLine(line)
			if lineErr != nil {
				err = kerrs.Append(err, lineErr)
				continue
			}

			// NOT
			// if err != nil {
			//	return
			// }
		}
		return
	}

	err3 := handleCSV("a\n1234\nb11111\nc")
	fmt.Printf("%+v\n", err3)

	// Output:
	// 2 errors occurred:
	// 	* Invalid Length for 1234
	// 	* Invalid Length for b11111
}

func ExampleExtract_errors() {
	err0 := errors.New("hi, I am an error")
	err1 := kerrs.Wrapv(err0, "wrong", "code", "12123", "value", 12312)
	err2 := kerrs.Wrapv(err1, "more explain about the error", "product_name", "iphone", "color", "red")
	err3 := kerrs.Wrapv(err2, "in regexp", "request_id", "T1212123129983")
	kvs, msg, stacktrace := kerrs.Extract(err3)

	var actual = bytes.NewBuffer(nil)
	fmt.Fprintln(actual, "\nmsg:", msg)
	fmt.Fprintf(actual, "\nkeyvals: %#+v\n\n", kvs)
	fmt.Fprintf(actual, "stacktrace:\n%s", cleanStacktrace(stacktrace))

	expected := `
msg: in regexp: more explain about the error: wrong: hi, I am an error

keyvals: []interface {}{"request_id", "T1212123129983", "product_name", "iphone", "color", "red", "code", "12123", "value", 12312}

stacktrace:
in regexp request_id=T1212123129983: more explain about the error product_name=iphone color=red: wrong code=12123 value=12312: hi, I am an error
<stacktrace>
`

	diff := testingutils.PrettyJsonDiff(expected, actual.String())
	fmt.Println(diff)
	// Output:
	//

}

func cleanStacktrace(stacktrace string) (cleantrace string) {
	cleantrace = strings.Split(stacktrace, "\n")[0]
	cleantrace = cleantrace + "\n<stacktrace>\n"
	return
}
