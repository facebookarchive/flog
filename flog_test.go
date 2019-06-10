// Package flog is a hacked and slashed version of glog that only logs in stderr
// and can be configured with env vars.
//
// Copyright 2019-present Facebook Inc. All Rights Reserved.
// Copyright 2013 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
package flog

import (
	"bytes"
	"fmt"
	stdLog "log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// newBuffers sets the log writers to all new byte buffers and returns the old array.
func (l *loggingT) newBuffers() {
	l.out = &bytes.Buffer{}
}

func (l *loggingT) revertBuffer() {
	l.out = os.Stderr
}

// contents returns the specified log value as a string.
func contents() string {
	if buf, ok := logging.out.(*bytes.Buffer); ok {
		return buf.String()
	}
	return ""
}

// contains reports whether the string is contained in the log.
func contains(str string) bool {
	return strings.Contains(contents(), str)
}

// Test that SetOutput works
func TestSetOutput(t *testing.T) {
	b := new(bytes.Buffer)
	SetOutput(b)
	defer SetOutput(os.Stderr)
	Info("test")
	if !contains("test") {
		t.Errorf("SetOuput failed!")
	}
}

// Ensure that Info, Infof and Infoln work.
func TestInfo(t *testing.T) {
	testLevel(t, "Info", Info, Infof, Infoln, InfoDepth)
}

func init() {
	CopyStandardLogTo("INFO")
}

// Test that CopyStandardLogTo panics on bad input.
func TestCopyStandardLogToPanic(t *testing.T) {
	defer func() {
		if s, ok := recover().(string); !ok || !strings.Contains(s, "LOG") {
			t.Errorf(`CopyStandardLogTo("LOG") should have panicked: %v`, s)
		}
	}()
	CopyStandardLogTo("LOG")
}

// Test that using the standard log package logs to INFO.
func TestStandardLog(t *testing.T) {
	logging.newBuffers()
	defer logging.revertBuffer()
	stdLog.Print("test")
	if !contains("I") {
		logging.revertBuffer()
		t.Errorf("Info has wrong character: %q", contents())
	}
	if !contains("test") {
		logging.revertBuffer()
		t.Error("Info failed")
	}
}

// Test that the header has the correct format.
func TestHeader(t *testing.T) {
	logging.newBuffers()
	defer logging.revertBuffer()
	defer func(previous func() time.Time) { timeNow = previous }(timeNow)
	timeNow = func() time.Time {
		return time.Date(2006, 1, 2, 15, 4, 5, .067890e9, time.Local)
	}
	pid = 1234
	Info("test")
	var line int
	format := "I0102 15:04:05.067890    1234 flog_test.go:%d] test\n"
	n, err := fmt.Sscanf(contents(), format, &line)
	if n != 1 || err != nil {
		logging.revertBuffer()
		t.Errorf("log format error: %d elements, error %s:\n%s", n, err, contents())
	}
	// Scanf treats multiple spaces as equivalent to a single space,
	// so check for correct space-padding also.
	want := fmt.Sprintf(format, line)
	if contents() != want {
		logging.revertBuffer()
		t.Errorf("log format error: got:\n\t%q\nwant:\t%q", contents(), want)
	}
}

// Ensure that Critical, Criticalf and Criticalln work.
func TestCritical(t *testing.T) {
	testLevel(t, "Critical", Critical, Criticalf, Criticalln, CriticalDepth)
}

// Ensure that Error, Errorf and Errorln work.
func TestError(t *testing.T) {
	testLevel(t, "Error", Error, Errorf, Errorln, ErrorDepth)
}

// Ensure that Warning, Warningf and Warningln work.
func TestWarning(t *testing.T) {
	testLevel(t, "Warning", Warning, Warningf, Warningln, WarningDepth)
}

// Ensure that Debug, Debugf and Debugln work.
func TestDebug(t *testing.T) {
	testLevel(t, "Debug", Debug, Debugf, Debugln, DebugDepth)
}

// For level X (where X is Debug, Info, ...), test the functions X, Xf, Xln and
// XDepth.
func testLevel(
	t *testing.T,
	name string,
	x func(...interface{}),
	xf func(string, ...interface{}),
	xln func(...interface{}),
	xDepth func(int, ...interface{}),
) {
	logging.newBuffers()
	defer logging.revertBuffer()

	x("Zaphod Beeblebrox")
	ensureInLog(t, name, "Zaphod Beeblebrox")

	logging.newBuffers()
	xf("%s is g%03x", "Go", 13)
	ensureInLog(t, name+"f", "Go is g00d")

	logging.newBuffers()
	xln("Rumpelstiltskin")
	ensureInLog(t, name+"ln", "Rumpelstiltskin")

	logging.newBuffers()
	testXDepth(t, name+"Depth", xDepth)
}

// Determine if a message is found in the appropriate log with the correct
// severity level
func ensureInLog(t *testing.T, name string, phrase string) {
	if !contains(nameToCode(name)) || !contains(phrase) {
		logging.revertBuffer()
		t.Errorf("%s failed", name)
	}
}

// For level X (where X is Debug, Info, ...), test the XDepth function.
func testXDepth(t *testing.T, name string, xDepth func(int, ...interface{})) {
	f := func() { xDepth(1, "depth-test1") }

	// The next three lines must stay together
	_, _, wantLine, _ := runtime.Caller(0)
	xDepth(0, "depth-test0")
	f()

	msgs := strings.Split(strings.TrimSuffix(contents(), "\n"), "\n")
	if len(msgs) != 2 {
		logging.revertBuffer()
		t.Fatalf("Got %d lines, expected 2", len(msgs))
	}

	for i, m := range msgs {
		if !strings.HasPrefix(m, nameToCode(name)) {
			logging.revertBuffer()
			t.Errorf("%s[%d] has wrong character: %q", name, i, m)
		}
		w := fmt.Sprintf("depth-test%d", i)
		if !strings.Contains(m, w) {
			logging.revertBuffer()
			t.Errorf("%s[%d] missing %q: %q", name, i, w, m)
		}

		// pull out the line number (between : and ])
		msg := m[strings.LastIndex(m, ":")+1:]
		x := strings.Index(msg, "]")
		if x < 0 {
			logging.revertBuffer()
			t.Errorf("%s[%d]: missing ']': %q", name, i, m)
			continue
		}
		line, err := strconv.Atoi(msg[:x])
		if err != nil {
			logging.revertBuffer()
			t.Errorf("%s[%d]: bad line number: %q", name, i, m)
			continue
		}
		wantLine++
		if wantLine != line {
			logging.revertBuffer()
			t.Errorf("%s[%d]: got line %d, want %d", name, i, line, wantLine)
		}
	}
}

// Convert a logging level name to the its one character code.  Codes are:
//   V: DEBUG
//   I: INFO
//   W: WARNING
//   E: ERROR
//   C: CRITICAL
//   F: FATAL
func nameToCode(name string) string {
	code := name[:1]
	if code == "D" {
		code = "V"
	}
	return code
}

// Test that a V log goes to Info.
func TestV(t *testing.T) {
	logging.newBuffers()
	defer logging.revertBuffer()
	logging.verbosity.Set("2")
	defer logging.verbosity.Set("0")
	V(2).Info("test")
	if !contains("I") {
		logging.revertBuffer()
		t.Errorf("Info has wrong character: %q", contents())
	}
	if !contains("test") {
		logging.revertBuffer()
		t.Error("Info failed")
	}
}

// Test that a config sets the V value correctly
func TestConfigV(t *testing.T) {
	logging.newBuffers()
	defer logging.revertBuffer()
	cfg := &Config{
		Verbosity: "2",
	}
	defer logging.verbosity.Set("0")
	if err := cfg.Set(); err != nil {
		logging.revertBuffer()
		t.Errorf("Failed to set parameters: %v", err)
	}
	V(2).Info("test")
	if !contains("I") {
		logging.revertBuffer()
		t.Errorf("Info has wrong character: %q", contents())
	}
	if !contains("test") {
		logging.revertBuffer()
		t.Error("Info failed")
	}
}

// Test that a vmodule enables a log in this file.
func TestVmoduleOn(t *testing.T) {
	logging.newBuffers()
	defer logging.revertBuffer()
	logging.vmodule.Set("flog_test=2")
	defer logging.vmodule.Set("")
	if !V(1) {
		logging.revertBuffer()
		t.Error("V not enabled for 1")
	}
	if !V(2) {
		logging.revertBuffer()
		t.Error("V not enabled for 2")
	}
	if V(3) {
		logging.revertBuffer()
		t.Error("V enabled for 3")
	}
	V(2).Info("test")
	if !contains("I") {
		logging.revertBuffer()
		t.Errorf("Info has wrong character: %q", contents())
	}
	if !contains("test") {
		logging.revertBuffer()
		t.Error("Info failed")
	}
}

// Test that a vmodule of another file does not enable a log in this file.
func TestVmoduleOff(t *testing.T) {
	logging.newBuffers()
	defer logging.revertBuffer()
	logging.vmodule.Set("notthisfile=2")
	defer logging.vmodule.Set("")
	for i := 1; i <= 3; i++ {
		if V(Level(i)) {
			logging.revertBuffer()
			t.Errorf("V enabled for %d", i)
		}
	}
	V(2).Info("test")
	if contents() != "" {
		logging.revertBuffer()
		t.Error("V logged incorrectly")
	}
}

// vGlobs are patterns that match/don't match this file at V=2.
var vGlobs = map[string]bool{
	// Easy to test the numeric match here.
	"flog_test=1": false, // If -vmodule sets V to 1, V(2) will fail.
	"flog_test=2": true,
	"flog_test=3": true, // If -vmodule sets V to 1, V(3) will succeed.
	// These all use 2 and check the patterns. All are true.
	"*=2":           true,
	"?l*=2":         true,
	"????_*=2":      true,
	"??[mno]?_*t=2": true,
	// These all use 2 and check the patterns. All are false.
	"*x=2":         false,
	"m*=2":         false,
	"??_*=2":       false,
	"?[abc]?_*t=2": false,
}

// Test that vmodule globbing works as advertised.
func testVmoduleGlob(pat string, match bool, t *testing.T) {
	logging.newBuffers()
	defer logging.revertBuffer()
	defer logging.vmodule.Set("")
	logging.vmodule.Set(pat)
	if V(2) != Verbose(match) {
		logging.revertBuffer()
		t.Errorf("incorrect match for %q: got %t expected %t", pat, V(2), match)
	}
}

// Test that a vmodule globbing works as advertised.
func TestVmoduleGlob(t *testing.T) {
	for glob, match := range vGlobs {
		testVmoduleGlob(glob, match, t)
	}
}

func TestLogBacktraceAt(t *testing.T) {
	logging.newBuffers()
	defer logging.revertBuffer()
	// The peculiar style of this code simplifies line counting and maintenance of the
	// tracing block below.
	var infoLine string
	setTraceLocation := func(file string, line int, ok bool, delta int) {
		if !ok {
			t.Fatal("could not get file:line")
		}
		_, file = filepath.Split(file)
		infoLine = fmt.Sprintf("%s:%d", file, line+delta)
		err := logging.traceLocation.Set(infoLine)
		if err != nil {
			logging.revertBuffer()
			t.Fatal("error setting log_backtrace_at: ", err)
		}
	}
	{
		// Start of tracing block. These lines know about each other's relative position.
		_, file, line, ok := runtime.Caller(0)
		setTraceLocation(file, line, ok, +2) // Two lines between Caller and Info calls.
		Info("we want a stack trace here")
	}
	numAppearances := strings.Count(contents(), infoLine)
	if numAppearances < 2 {
		// Need 2 appearances, one in the log header and one in the trace:
		//   log_test.go:281: I0511 16:36:06.952398 02238 log_test.go:280] we want a stack trace here
		//   ...
		//   github.com/glog/glog_test.go:280 (0x41ba91)
		//   ...
		// We could be more precise but that would require knowing the details
		// of the traceback format, which may not be dependable.
		logging.revertBuffer()
		t.Fatal("got no trace back; log is ", contents())
	}
}

func TestGetVerbosity(t *testing.T) {
	logging.verbosity.Set("5")
	defer logging.verbosity.Set("0")
	v := GetVerbosity()
	if v != 5 {
		t.Fatalf("invalid verbosity: want: 5, got %d", v)
	}
}

func BenchmarkHeader(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf, _, _ := logging.header(infoLog, 0)
		logging.putBuffer(buf)
	}
}

func BenchmarkHeaderParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf, _, _ := logging.header(infoLog, 0)
			logging.putBuffer(buf)
		}
	})
}
