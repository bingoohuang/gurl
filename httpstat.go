package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http/httptrace"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
)

// some code is copy from https://github.com/davecheney/httpstat.

func printf(format string, a ...interface{}) {
	_, _ = fmt.Fprintf(color.Output, format, a...)
}

func grayscale(code color.Attribute) func(string, ...interface{}) string {
	return color.New(code + 232).SprintfFunc()
}

type httpStat struct {
	t0, t1, t2, t3, t4, t5, t6 time.Time
	t7                         time.Time // after read body
}

func createClientTrace(req *Request) *httptrace.ClientTrace {
	stat := &httpStat{}
	req.stat = stat
	return &httptrace.ClientTrace{
		DNSStart: func(_ httptrace.DNSStartInfo) {
			stat.t0 = time.Now()
		},
		DNSDone: func(_ httptrace.DNSDoneInfo) {
			stat.t1 = time.Now()
		},
		ConnectStart: func(_, _ string) {
			if stat.t1.IsZero() {
				stat.t1 = time.Now() // connecting to IP
			}
		},
		ConnectDone: func(net, addr string, err error) {
			if err != nil {
				log.Fatalf("unable to connect to host %v: %v", addr, err)
			}
			stat.t2 = time.Now()

			if HasPrintOption(printVerbose) {
				printf("\n%s%s\n", color.GreenString("Connected to "), color.CyanString(addr))
			}
		},
		GotConn: func(info httptrace.GotConnInfo) {
			stat.t3 = time.Now()
			req.ConnInfo = info
		},
		GotFirstResponseByte: func() {
			stat.t4 = time.Now()
		},
		TLSHandshakeStart: func() {
			stat.t5 = time.Now()
		},
		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			stat.t6 = time.Now()
		},
	}
}

func (stat *httpStat) print(urlSchema string) {
	now := time.Now()
	stat.t7 = now
	if stat.t0.IsZero() { // we skipped DNS
		stat.t0 = stat.t1
	}
	fmta := func(b, a time.Time) string {
		return color.CyanString("%7d ms", int(b.Sub(a)/time.Millisecond))
	}
	fmtb := func(b, a time.Time) string {
		return color.CyanString("%-9s", strconv.Itoa(int(b.Sub(a)/time.Millisecond))+" ms")
	}

	colorize := func(s string) string {
		v := strings.Split(s, "\n")
		v[0] = grayscale(16)(v[0])
		return strings.Join(v, "\n")
	}

	fmt.Println()

	switch urlSchema {
	case "https":
		printf(colorize(httpsTemplate),
			fmta(stat.t1, stat.t0), // dns lookup
			fmta(stat.t2, stat.t1), // tcp connection
			fmta(stat.t6, stat.t5), // tls handshake
			fmta(stat.t4, stat.t3), // server processing
			fmta(stat.t7, stat.t4), // content transfer
			fmtb(stat.t1, stat.t0), // namelookup
			fmtb(stat.t2, stat.t0), // connect
			fmtb(stat.t3, stat.t0), // pretransfer
			fmtb(stat.t4, stat.t0), // starttransfer
			fmtb(stat.t7, stat.t0), // total
		)
	case "http":
		printf(colorize(httpTemplate),
			fmta(stat.t1, stat.t0), // dns lookup
			fmta(stat.t3, stat.t1), // tcp connection
			fmta(stat.t4, stat.t3), // server processing
			fmta(stat.t7, stat.t4), // content transfer
			fmtb(stat.t1, stat.t0), // namelookup
			fmtb(stat.t3, stat.t0), // connect
			fmtb(stat.t4, stat.t0), // starttransfer
			fmtb(stat.t7, stat.t0), // total
		)
	}
}

func isRedirect(statusCode int) bool { return statusCode > 299 && statusCode < 400 }

const (
	httpsTemplate = `` +
		`  DNS Lookup   TCP Connection   TLS Handshake   Server Processing   Content Transfer` + "\n" +
		`[%s  |     %s |    %s |        %s |       %s ]` + "\n" +
		`             |                |               |                   |                  |` + "\n" +
		` namelookup: %s        |               |                   |                  |` + "\n" +
		`                     connect: %s       |                   |                  |` + "\n" +
		`                                 pretransfer: %s           |                  |` + "\n" +
		`                                                   starttransfer: %s          |` + "\n" +
		`                                                                              total: %s` + "\n"

	httpTemplate = `` +
		`   DNS Lookup   TCP Connection   Server Processing   Content Transfer` + "\n" +
		`[ %s  |     %s  |        %s |       %s]` + "\n" +
		`              |                 |                   |                 |` + "\n" +
		`  namelookup: %s         |                   |                 |` + "\n" +
		`                       connect: %s           |                 |` + "\n" +
		`                                     starttransfer: %s         |` + "\n" +
		`                                                               total: %s` + "\n"
)
