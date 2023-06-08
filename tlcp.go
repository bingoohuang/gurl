package main

import (
	"fmt"
	"net"
	"strings"

	"gitee.com/Trisia/gotlcp/tlcp"
	"github.com/bingoohuang/gg/pkg/osx"
	"github.com/bingoohuang/gg/pkg/osx/env"
	"github.com/emmansun/gmsm/smx509"
)

var tlcpSessionCache tlcp.SessionCache

func init() {
	if cacheSize := env.Int(`TLS_SESSION_CACHE`, 32); cacheSize > 0 {
		tlcpSessionCache = tlcp.NewLRUSessionCache(cacheSize)
	}
}

func createTlcpDialer(dialer *net.Dialer, caFile, tlcpCerts string) DialContextFn {
	c := &tlcp.Config{
		InsecureSkipVerify: !env.Bool(`TLS_VERIFY`, false),
		SessionCache:       tlcpSessionCache,
	}

	c.EnableDebug = HasPrintOption(printDebug)

	if caFile != "" {
		rootCert, err := smx509.ParseCertificatePEM(osx.ReadFile(caFile, osx.WithFatalOnError(true)).Data)
		if err != nil {
			panic(err)
		}
		pool := smx509.NewCertPool()
		pool.AddCert(rootCert)
		c.RootCAs = pool
	}

	if tlcpCerts != "" {
		// TLCP 1.1，套件ECDHE-SM2-SM4-CBC-SM3，设置客户端双证书
		certsFiles := strings.Split(tlcpCerts, ",")
		var certs []tlcp.Certificate
		switch len(certsFiles) {
		case 0, 2, 4:
		default:
			panic("-tclp-certs should be sign.cert.pem,sign.key.pem,enc.cert.pem,enc.key.pem")
		}
		if len(certs) >= 2 {
			signCertKeypair, err := tlcp.X509KeyPair(osx.ReadFile(certsFiles[0], osx.WithFatalOnError(true)).Data,
				osx.ReadFile(certsFiles[1], osx.WithFatalOnError(true)).Data)
			if err != nil {
				panic(err)
			}
			certs = append(certs, signCertKeypair)
		}
		if len(certs) >= 4 {
			encCertKeypair, err := tlcp.X509KeyPair(osx.ReadFile(certsFiles[2], osx.WithFatalOnError(true)).Data,
				osx.ReadFile(certsFiles[3], osx.WithFatalOnError(true)).Data)
			if err != nil {
				panic(err)
			}
			certs = append(certs, encCertKeypair)
		}

		if len(certs) > 0 {
			c.Certificates = certs
			c.CipherSuites = []uint16{tlcp.ECDHE_SM4_CBC_SM3, tlcp.ECDHE_SM4_GCM_SM3}
		}
	}

	d := tlcp.Dialer{NetDialer: dialer, Config: c}
	return d.DialContext
}

func printTLCPConnectState(state tlcp.ConnectionState) {
	if !HasPrintOption(printRspOption) {
		return
	}

	tlsVersion := func(version uint16) string {
		switch version {
		case tlcp.VersionTLCP:
			return "TLCP"
		default:
			return "Unknown"
		}
	}(state.Version)
	fmt.Printf("option TLCP.Version: %s\n", tlsVersion)
	fmt.Printf("option TLCP.HandshakeComplete: %t\n", state.HandshakeComplete)
	fmt.Printf("option TLCP.DidResume: %t\n", state.DidResume)
	fmt.Println()
}
