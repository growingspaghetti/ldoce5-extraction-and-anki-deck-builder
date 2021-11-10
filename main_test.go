package main

import (
	"strings"
	"testing"
)

var (
	l []string = []string{"The", "first", "line", "of", "the", "result,", "PASS,", "comes", "from", "the", "testing", "portion", "of", "the", "test", "driver,", "asking", "go", "test", "to", "run", "your", "benchmarks", "does", "not", "disable", "the", "tests", "in", "the", "package.", "If", "you", "want", "to", "skip", "the", "tests,", "you", "can", "do", "so", "by", "passing", "a", "regex", "to", "the", "-run", "flag", "that", "will", "not", "match", "anything.", "I", "usually", "use"}
	p pc       = pc{
		sb: new(strings.Builder),
	}
)

type pc struct {
	sb *strings.Builder
}

func (c pc) valuePathCat(dirs ...string) string {
	c.sb.Reset()
	for i := len(dirs) - 1; i >= 0; i-- {
		c.sb.WriteString("/")
		c.sb.WriteString(dirs[i])
	}
	return c.sb.String()
}

func (c *pc) referencePathCat(dirs ...string) string {
	c.sb.Reset()
	for i := len(dirs) - 1; i >= 0; i-- {
		c.sb.WriteString("/")
		c.sb.WriteString(dirs[i])
	}
	return c.sb.String()
}

func pathCat(dirs ...string) string {
	var sb strings.Builder
	for i := len(dirs) - 1; i >= 0; i-- {
		sb.WriteString("/")
		sb.WriteString(dirs[i])
	}
	return sb.String()
}

func pathCatString(dirs ...string) string {
	var s string
	for i := len(dirs) - 1; i >= 0; i-- {
		s += "/"
		s += dirs[i]
	}
	return s
}

// Running tool: /usr/local/go/bin/go test -benchmem -run=^$ -bench ^BenchmarkEach$ growingspaghetti/gauld-lang-syne -v

// goos: linux
// goarch: amd64
// pkg: growingspaghetti/gauld-lang-syne
// cpu: Intel(R) Core(TM) i7-3632QM CPU @ 2.20GHz
// BenchmarkEach
// BenchmarkEach-8   	1000000000	         0.01960 ns/op	       0 B/op	       0 allocs/op
// PASS
// ok  	growingspaghetti/gauld-lang-syne	0.169s
func BenchmarkEach(b *testing.B) {
	for i := 0; i < 20000; i++ {
		pathCat(l...)
	}
}

// Running tool: /usr/local/go/bin/go test -benchmem -run=^$ -bench ^BenchmarkReceiver$ growingspaghetti/gauld-lang-syne -v

// goos: linux
// goarch: amd64
// pkg: growingspaghetti/gauld-lang-syne
// cpu: Intel(R) Core(TM) i7-3632QM CPU @ 2.20GHz
// BenchmarkReceiver
// BenchmarkReceiver-8
// 1000000000	         0.02841 ns/op	       0 B/op	       0 allocs/op
// PASS
// ok  	growingspaghetti/gauld-lang-syne	0.185s
func BenchmarkReceiver(b *testing.B) {
	for i := 0; i < 20000; i++ {
		p.referencePathCat(l...)
	}
}

// Running tool: /usr/local/go/bin/go test -benchmem -run=^$ -bench ^BenchmarkValue$ growingspaghetti/gauld-lang-syne -v

// goos: linux
// goarch: amd64
// pkg: growingspaghetti/gauld-lang-syne
// cpu: Intel(R) Core(TM) i7-3632QM CPU @ 2.20GHz
// BenchmarkValue
// BenchmarkValue-8   	1000000000	         0.03512 ns/op	       0 B/op	       0 allocs/op
// PASS
// ok  	growingspaghetti/gauld-lang-syne	0.251s
func BenchmarkValue(b *testing.B) {
	for i := 0; i < 20000; i++ {
		p.valuePathCat(l...)
	}
}

// Running tool: /usr/local/go/bin/go test -benchmem -run=^$ -bench ^BenchmarkString$ growingspaghetti/gauld-lang-syne -v

// goos: linux
// goarch: amd64
// pkg: growingspaghetti/gauld-lang-syne
// cpu: Intel(R) Core(TM) i7-3632QM CPU @ 2.20GHz
// BenchmarkString
// BenchmarkString-8   	1000000000	         0.3610 ns/op	       0 B/op	       0 allocs/op
// PASS
// ok  	growingspaghetti/gauld-lang-syne	4.400s
func BenchmarkString(b *testing.B) {
	for i := 0; i < 20000; i++ {
		pathCatString(l...)
	}
}
