# 国密测试

1. 单向 TLS  `gurl https://192.168.136.114/v Host:www.xyz.com -pao -dn`
2. 单向 TLCP 双证 `TLCP=1 gurl https://192.168.136.114/v Host:www.xyz.com -pao -dn`
3. 双向 TLCP 双证 `TLCP_CERTS=sm2_client_sign.crt,sm2_client_sign.key,sm2_client_enc.crt,sm2_client_enc.key TLCP=1 gurl https://192.168.136.114/v Host:www.abc.com -pao -dn`

```sh
[root@jmjs-PC:/home/zys/tlcp]# TLCP=1 gurl https://192.168.136.114/v Host:www.xyz.com -pao -dn
option TLCP.Version: TLCP
option TLCP.Subject: CN=sm2_server_abc,OU=BJCA,O=BJCA,ST=BeiJing,C=CN
option TLCP.KeyUsage: [DigitalSignature ContentCommitment]
option TLCP.Subject: CN=sm2_server_abc,OU=BJCA,O=BJCA,ST=BeiJing,C=CN
option TLCP.KeyUsage: [KeyEncipherment DataEncipherment KeyAgreement]
option TLCP.HandshakeComplete: true
option TLCP.DidResume: false
option TLCP.CipherSuite: &{ID:57361 Name:ECDHE_SM4_CBC_SM3 SupportedVersions:[257] Insecure:false}
Conn-Session: 192.168.136.114:43860->192.168.136.114:443 (reused: false, wasIdle: false, idle: 0s)
GET /v HTTP/1.1
Host: www.xyz.com
Accept: */*
Accept-Encoding: gzip, deflate
Gurl-Date: Wed, 17 Jan 2024 05:28:47 GMT
User-Agent: curl/1.0.0


HTTP/1.1 200 OK
Server: Tengine/2.4.0
Date: Wed, 17 Jan 2024 05:28:47 GMT
Content-Type: application/octet-stream
Content-Length: 106
Connection: keep-alive

www.xyz.com 单向SSL通信 test OK, ssl_protocol is NTLSv1.1 (NTLSv1.1 表示国密，其他表示国际)
2024/01/17 13:28:47.247972 main.go:96: complete, total cost: 5.989214ms
[root@jmjs-PC:/home/zys/tlcp]#
[root@jmjs-PC:/home/zys/tlcp]#
[root@jmjs-PC:/home/zys/tlcp]# gurl https://192.168.136.114/v Host:www.xyz.com -pao -dn
option TLS.Version: TLSv13
option TLS.Subject: CN=ecc_server_abc,OU=BJCA,O=BJCA,ST=BeiJing,C=CN
option TLS.KeyUsage: [DigitalSignature ContentCommitment]
option TLS.HandshakeComplete: true
option TLS.DidResume: false
option TLS.CipherSuite: &{ID:4866 Name:TLS_AES_256_GCM_SHA384 SupportedVersions:[772] Insecure:false}
Conn-Session: 192.168.136.114:43862->192.168.136.114:443 (reused: false, wasIdle: false, idle: 0s)
GET /v HTTP/1.1
Host: www.xyz.com
Accept: */*
Accept-Encoding: gzip, deflate
Gurl-Date: Wed, 17 Jan 2024 05:28:53 GMT
User-Agent: curl/1.0.0


HTTP/1.1 200 OK
Connection: keep-alive
Server: Tengine/2.4.0
Date: Wed, 17 Jan 2024 05:28:53 GMT
Content-Type: application/octet-stream
Content-Length: 105

www.xyz.com 单向SSL通信 test OK, ssl_protocol is TLSv1.3 (NTLSv1.1 表示国密，其他表示国际)
2024/01/17 13:28:53.323278 main.go:96: complete, total cost: 2.12606ms
[root@jmjs-PC:/home/zys/tlcp]#
[root@jmjs-PC:/home/zys/tlcp]#
[root@jmjs-PC:/home/zys/tlcp]# hostname -I
192.168.136.114
[root@jmjs-PC:/home/zys/tlcp]#
[root@jmjs-PC:/home/zys/tlcp]# hostname
jmjs-PC
[root@jmjs-PC:/home/zys/tlcp]# uname -a
Linux jmjs-PC 4.19.0-arm64-server #1707 SMP Thu Mar 26 17:43:52 CST 2020 aarch64 GNU/Linux
```

```shell
[root@jmjs-PC:/home/zys/tlcp]# cd /home/zys/tlcp
[root@jmjs-PC:/home/zys/tlcp]# export TLCP_CERTS=sm2_client_sign.crt,sm2_client_sign.key,sm2_client_enc.crt,sm2_client_enc.key
[root@jmjs-PC:/home/zys/tlcp]# TLCP=1 gurl https://192.168.136.114/v Host:www.abc.com -pao -dn
option TLCP.Version: TLCP
option TLCP.Subject: CN=sm2_server_abc,OU=BJCA,O=BJCA,ST=BeiJing,C=CN
option TLCP.KeyUsage: [DigitalSignature ContentCommitment]
option TLCP.Subject: CN=sm2_server_abc,OU=BJCA,O=BJCA,ST=BeiJing,C=CN
option TLCP.KeyUsage: [KeyEncipherment DataEncipherment KeyAgreement]
option TLCP.HandshakeComplete: true
option TLCP.DidResume: false
option TLCP.CipherSuite: &{ID:57361 Name:ECDHE_SM4_CBC_SM3 SupportedVersions:[257] Insecure:false}
Conn-Session: 192.168.136.114:43832->192.168.136.114:443 (reused: false, wasIdle: false, idle: 0s)
GET /v HTTP/1.1
Host: www.abc.com
Accept: */*
Accept-Encoding: gzip, deflate
Gurl-Date: Wed, 17 Jan 2024 05:22:38 GMT
User-Agent: curl/1.0.0


HTTP/1.1 200 OK
Connection: keep-alive
Server: Tengine/2.4.0
Date: Wed, 17 Jan 2024 05:22:38 GMT
Content-Type: application/octet-stream
Content-Length: 106

www.abc.com 双向SSL通信 test OK, ssl_protocol is NTLSv1.1 (NTLSv1.1 表示国密，其他表示国际)
2024/01/17 13:22:38.542861 main.go:96: complete, total cost: 6.165817ms
[root@jmjs-PC:/home/zys/tlcp]#
[root@jmjs-PC:/home/zys/tlcp]#
[root@jmjs-PC:/home/zys/tlcp]#
[root@jmjs-PC:/home/zys/tlcp]# TLCP=1 gurl https://192.168.136.114/v Host:www.abc.com -paod -dn
load 2 client certs
[write] Client Hello, len=47, success=true
>>> ClientHello
Random: bytes=65a764652a11f953780a8a364423ee2cd87bc61c6b0e22158a5c778c644c6a6a
Session ID:
Cipher Suites: ECDHE_SM4_GCM_SM3, ECDHE_SM4_CBC_SM3,
Compression Methods: [0]
<<<
[read] Server Hello, len=74
>>> ServerHello
Random: bytes=f39c90fadc7d5ffa337ab231971de80e2d4f5756686c379a444f574e47524400
Session ID: 7c11e747a38b6a8069da4ffd6e56a0ff6623610ee97e148be5f1b18275992787
Cipher Suite: ECDHE_SM4_CBC_SM3
Compression Method: 0
<<<
[read] Certificate, len=960
>>> Certificates
Cert[0]:
-----BEGIN CERTIFICATE-----
MIIB1TCCAXygAwIBAgIKMAAAAAAAAAAAAzAKBggqgRzPVQGDdTBWMQswCQYDVQQG
EwJDTjEQMA4GA1UECAwHQmVpSmluZzENMAsGA1UECgwEQkpDQTENMAsGA1UECwwE
QkpDQTEXMBUGA1UEAwwOc20yX3Rlc3Rfc3ViY2EwHhcNMjQwMTE0MDU0NDA5WhcN
MzQwMTExMDU0NDA5WjBWMQswCQYDVQQGEwJDTjEQMA4GA1UECAwHQmVpSmluZzEN
MAsGA1UECgwEQkpDQTENMAsGA1UECwwEQkpDQTEXMBUGA1UEAwwOc20yX3NlcnZl
cl9hYmMwWTATBgcqhkjOPQIBBggqgRzPVQGCLQNCAAQkv1y/TidHO0yk8FZ5OzEQ
rZnc6/kzlIG/blaqFM9/RALB/yjSg2RBt1+1iixEtAPaVvwPBGwRGWqqTR5RFsb8
ozIwMDAJBgNVHRMEAjAAMAsGA1UdDwQEAwIGwDAWBgNVHREEDzANggt3d3cuYWJj
LmNvbTAKBggqgRzPVQGDdQNHADBEAiAdCdgX82Fqa3LMHyBJSZdLRd/cDwXMbrWb
7GTCkJKWbgIgZRe7oDobL6DUrWz3xg2AGQHVvpX2aDgOCirKqPYMt+c=
-----END CERTIFICATE-----
Cert[1]:
-----BEGIN CERTIFICATE-----
MIIB1jCCAXygAwIBAgIKMAAAAAAAAAAABDAKBggqgRzPVQGDdTBWMQswCQYDVQQG
EwJDTjEQMA4GA1UECAwHQmVpSmluZzENMAsGA1UECgwEQkpDQTENMAsGA1UECwwE
QkpDQTEXMBUGA1UEAwwOc20yX3Rlc3Rfc3ViY2EwHhcNMjQwMTE0MDU0NDA5WhcN
MzQwMTExMDU0NDA5WjBWMQswCQYDVQQGEwJDTjEQMA4GA1UECAwHQmVpSmluZzEN
MAsGA1UECgwEQkpDQTENMAsGA1UECwwEQkpDQTEXMBUGA1UEAwwOc20yX3NlcnZl
cl9hYmMwWTATBgcqhkjOPQIBBggqgRzPVQGCLQNCAAR95HI5Kz8iLSGIiaOrcCH9
XMjfV860u84Xwk7TRR2TeYhTgKyARZVQMQ7WcqqsjDmcliboHwyYHaW91bLf6kPz
ozIwMDAJBgNVHRMEAjAAMAsGA1UdDwQEAwIDODAWBgNVHREEDzANggt3d3cuYWJj
LmNvbTAKBggqgRzPVQGDdQNIADBFAiBXJqiy7HHlLdNgGlwvXzgm+Ec1IJLxH8Bg
R5UEve+c8AIhAK0rOHfJaCmopHRvMbzmV/oDOkvdVMfSjaz1yZ06G0RV
-----END CERTIFICATE-----
<<<
[read] Server Key Exchange, len=146
[read] Certificate Request, len=546
>>> Certificate Request
Certificate Types: RSA, ECDSA
Certificate Authorities:
Issuer[0]:
CN=rsa_test_root,OU=BJCA,O=BJCA,ST=BeiJing,C=CN
Issuer[1]:
CN=rsa_test_subca,OU=BJCA,O=BJCA,ST=BeiJing,C=CN
Issuer[2]:
CN=ecc_test_root,OU=BJCA,O=BJCA,ST=BeiJing,C=CN
Issuer[3]:
CN=ecc_test_subca,OU=BJCA,O=BJCA,ST=BeiJing,C=CN
Issuer[4]:
CN=sm2_test_root,OU=BJCA,O=BJCA,ST=BeiJing,C=CN
Issuer[5]:
CN=sm2_test_subca,OU=BJCA,O=BJCA,ST=BeiJing,C=CN
<<<
[read] Server Hello Done, len=4
[write] Certificate, len=954, success=true
>>> Certificates
Cert[0]:
-----BEGIN CERTIFICATE-----
MIIB0zCCAXigAwIBAgIKMAAAAAAAAAAABzAKBggqgRzPVQGDdTBWMQswCQYDVQQG
EwJDTjEQMA4GA1UECAwHQmVpSmluZzENMAsGA1UECgwEQkpDQTENMAsGA1UECwwE
QkpDQTEXMBUGA1UEAwwOc20yX3Rlc3Rfc3ViY2EwHhcNMjQwMTE0MDU0NDA5WhcN
MzQwMTExMDU0NDA5WjBSMQswCQYDVQQGEwJDTjEQMA4GA1UECAwHQmVpSmluZzEN
MAsGA1UECgwEQkpDQTENMAsGA1UECwwEQkpDQTETMBEGA1UEAwwKc20yX2NsaWVu
dDBZMBMGByqGSM49AgEGCCqBHM9VAYItA0IABNG+XLyhLd/c33zP4oq9gOIt4YIK
bC+DmcvGL0uDkHW9/Dc8fSdX0L2C7WCgfIc60ApE5b9nDRpONwYl/bD9z4+jMjAw
MAkGA1UdEwQCMAAwCwYDVR0PBAQDAgbAMBYGA1UdEQQPMA2CC3d3dy54eXouY29t
MAoGCCqBHM9VAYN1A0kAMEYCIQCfu7RvIk1CBAVvRalRkMwynjQQcWLDyWyEK7ge
HMtHYwIhAOre8juLRq/Lvzkb3H4Zmn6Gf86aBhTiqbKcAk2/GO18
-----END CERTIFICATE-----
Cert[1]:
-----BEGIN CERTIFICATE-----
MIIB0jCCAXigAwIBAgIKMAAAAAAAAAAACTAKBggqgRzPVQGDdTBWMQswCQYDVQQG
EwJDTjEQMA4GA1UECAwHQmVpSmluZzENMAsGA1UECgwEQkpDQTENMAsGA1UECwwE
QkpDQTEXMBUGA1UEAwwOc20yX3Rlc3Rfc3ViY2EwHhcNMjQwMTE2MDYxMDIxWhcN
MzQwMTEzMDYxMDIxWjBSMQswCQYDVQQGEwJDTjEQMA4GA1UECAwHQmVpSmluZzEN
MAsGA1UECgwEQkpDQTENMAsGA1UECwwEQkpDQTETMBEGA1UEAwwKc20yX2NsaWVu
dDBZMBMGByqGSM49AgEGCCqBHM9VAYItA0IABBZM9ngYLQEBJa69Gx/4kYsvhoca
dasXSR1wEoP06t88GdymtFMbn0T4oTIfuOljSCPaZLVgNFinGgbdzC+ptvCjMjAw
MAkGA1UdEwQCMAAwCwYDVR0PBAQDAgM4MBYGA1UdEQQPMA2CC3d3dy54eXouY29t
MAoGCCqBHM9VAYN1A0gAMEUCIQCvPGO5WJ0W6GQpLPr2lpms4VXAxAd0L0PF0Jw6
NejcEwIgG9+13XWiYlGDwW0rr+h70aCnlNesnex1drQuSmtoocg=
-----END CERTIFICATE-----
<<<
[write] Client Key Exchange, len=73, success=true
[write] Certificate Verify, len=77, success=true
[write] Finished, len=16, success=true
>>> Finished
verify_data: [83 117 17 250 200 97 231 29 41 180 159 124]
<<<
[read] Finished, len=16
>>> Finished
verify_data: [226 224 239 47 24 55 199 27 248 130 238 45]
<<<
option TLCP.Version: TLCP
option TLCP.Subject: CN=sm2_server_abc,OU=BJCA,O=BJCA,ST=BeiJing,C=CN
option TLCP.KeyUsage: [DigitalSignature ContentCommitment]
option TLCP.Subject: CN=sm2_server_abc,OU=BJCA,O=BJCA,ST=BeiJing,C=CN
option TLCP.KeyUsage: [KeyEncipherment DataEncipherment KeyAgreement]
option TLCP.HandshakeComplete: true
option TLCP.DidResume: false
option TLCP.CipherSuite: &{ID:57361 Name:ECDHE_SM4_CBC_SM3 SupportedVersions:[257] Insecure:false}
Conn-Session: 192.168.136.114:43834->192.168.136.114:443 (reused: false, wasIdle: false, idle: 0s)
GET /v HTTP/1.1
Host: www.abc.com
Accept: */*
Accept-Encoding: gzip, deflate
Gurl-Date: Wed, 17 Jan 2024 05:23:49 GMT
User-Agent: curl/1.0.0


HTTP/1.1 200 OK
Content-Length: 106
Connection: keep-alive
Server: Tengine/2.4.0
Date: Wed, 17 Jan 2024 05:23:49 GMT
Content-Type: application/octet-stream

www.abc.com 双向SSL通信 test OK, ssl_protocol is NTLSv1.1 (NTLSv1.1 表示国密，其他表示国际)
2024/01/17 13:23:49.365895 main.go:96: complete, total cost: 6.434721ms
```