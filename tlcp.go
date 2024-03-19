package main

import (
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"strings"

	"gitee.com/Trisia/gotlcp/tlcp"
	"github.com/bingoohuang/gg/pkg/osx"
	"github.com/bingoohuang/gg/pkg/osx/env"
	"github.com/bingoohuang/gurl/certinfo"
	"github.com/emmansun/gmsm/smx509"
)

var tlcpSessionCache tlcp.SessionCache

func init() {
	if cacheSize := env.Int(`TLS_SESSION_CACHE`, 32); cacheSize > 0 {
		tlcpSessionCache = tlcp.NewLRUSessionCache(cacheSize)
	}
}

var tlcpCerts = os.Getenv("TLCP_CERTS")

func createTlcpDialer(dialer *net.Dialer, caFile string) DialContextFn {
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
			panic("$TLCP_CERTS should be sign.cert.pem,sign.key.pem,enc.cert.pem,enc.key.pem")
		}
		if len(certsFiles) >= 2 {
			signCertKeypair, err := tlcp.X509KeyPair(osx.ReadFile(certsFiles[0], osx.WithFatalOnError(true)).Data,
				osx.ReadFile(certsFiles[1], osx.WithFatalOnError(true)).Data)
			if err != nil {
				panic(err)
			}
			certs = append(certs, signCertKeypair)
		}
		if len(certsFiles) >= 4 {
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

	if c.EnableDebug {
		fmt.Printf("load %d client certs\n", len(c.Certificates))
	}

	d := tlcp.Dialer{NetDialer: dialer, Config: c}
	return d.DialContext
}

func printTLCPConnectState(conn net.Conn, state tlcp.ConnectionState) {
	if !HasPrintOption(printRspOption) {
		return
	}

	fmt.Printf("option Conn type: %T\n", conn)
	fmt.Printf("option TLCP.Version: %s\n", func(version uint16) string {
		switch version {
		case tlcp.VersionTLCP:
			return "TLCP"
		default:
			return "Unknown"
		}
	}(state.Version))
	for i, cert := range state.PeerCertificates {
		text, _ := certinfo.CertificateText((*x509.Certificate)(cert))
		fmt.Printf("option Cert[%d]: %s\n", i, text)
	}
	fmt.Printf("option TLCP.HandshakeComplete: %t\n", state.HandshakeComplete)
	fmt.Printf("option TLCP.DidResume: %t\n", state.DidResume)
	for _, suit := range tlcp.CipherSuites() {
		if suit.ID == state.CipherSuite {
			fmt.Printf("option TLCP.CipherSuite: %+v", suit)
			break
		}
	}
	fmt.Println()
}

// KeyUsageString convert x509.KeyUsage to string.
func KeyUsageString(k x509.KeyUsage) []string {
	var usages []string

	if k&x509.KeyUsageDigitalSignature != 0 {
		usages = append(usages, "DigitalSignature")
	}
	if k&x509.KeyUsageContentCommitment != 0 {
		usages = append(usages, "ContentCommitment")
	}
	if k&x509.KeyUsageKeyEncipherment != 0 {
		usages = append(usages, "KeyEncipherment")
	}
	if k&x509.KeyUsageDataEncipherment != 0 {
		usages = append(usages, "DataEncipherment")
	}
	if k&x509.KeyUsageKeyAgreement != 0 {
		usages = append(usages, "KeyAgreement")
	}
	if k&x509.KeyUsageCertSign != 0 {
		usages = append(usages, "CertSign")
	}
	if k&x509.KeyUsageCRLSign != 0 {
		usages = append(usages, "CRLSign")
	}
	if k&x509.KeyUsageEncipherOnly != 0 {
		usages = append(usages, "EncipherOnly")
	}
	if k&x509.KeyUsageDecipherOnly != 0 {
		usages = append(usages, "DecipherOnly")
	}

	return usages
}
