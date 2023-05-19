# changes

1. 2023年05月19日 文件上传时支持请求头 `Beefs-Hash: sm3:xxx`，用法 `BEEFS_HASH=sm3 gurl :9335 -auth scott:tiger -F stock-photo-1069484432.jpg` 
2. 2023年04月10日 支持 TLS SESSION REUSE

    ```shell
    # 1. 测试标准 SSL 连接，调用2次，打印 session 和 TLS 选项，可以看到，会话保持，只有 1 次握手
    $ gurl https://192.168.126.18:22443/ -pso -n2
    option TLS.Version: TLSv12
    option TLS.HandshakeComplete: true
    option TLS.DidResume: false
    
    Conn-Session: 2.0.1.1:61658->192.168.126.18:22443 (reused: false, wasIdle: false, idle: 0s)
    Conn-Session: 2.0.1.1:61658->192.168.126.18:22443 (reused: true, wasIdle: true, idle: 202.262µs)

    # 2. 测试标准 SSL 连接，调用2次，打印 session 和 TLS 选项，可以看到，会话不保持，有 2 次握手，但是 TLS 会话重用了
    $ gurl https://192.168.126.18:22443/ -pso -n2 -k
    option TLS.Version: TLSv12
    option TLS.HandshakeComplete: true
    option TLS.DidResume: false
    
    Conn-Session: 2.0.1.1:61734->192.168.126.18:22443 (reused: false, wasIdle: false, idle: 0s)
    option TLS.Version: TLSv12
    option TLS.HandshakeComplete: true
    option TLS.DidResume: true
    
    Conn-Session: 2.0.1.1:61735->192.168.126.18:22443 (reused: false, wasIdle: false, idle: 0s)
   
    # 3. 测试标准 SSL 连接，调用2次，打印 session 和 TLS 选项，可以看到，会话不保持，TLS 会话不重用（关闭重用缓存）
    $ TLS_SESSION_CACHE=0 gurl https://192.168.126.18:22443/ -pso -n2 -k
    option TLS.Version: TLSv12
    option TLS.HandshakeComplete: true
    option TLS.DidResume: false
    
    Conn-Session: 2.0.1.1:62056->192.168.126.18:22443 (reused: false, wasIdle: false, idle: 0s)
    option TLS.Version: TLSv12
    option TLS.HandshakeComplete: true
    option TLS.DidResume: false
    
    Conn-Session: 2.0.1.1:62057->192.168.126.18:22443 (reused: false, wasIdle: false, idle: 0s)
    ```
   
    ```sh
    # 对应到国密 TLCP 使用情况
    # 1
    $ gurl https://192.168.126.18:15443/ -pso -n2 -tlcp
    option TLCP.Version: TLCP
    option TLCP.HandshakeComplete: true
    option TLCP.DidResume: false
    
    Conn-Session: 2.0.1.1:62167->192.168.126.18:15443 (reused: false, wasIdle: false, idle: 0s)
    Conn-Session: 2.0.1.1:62167->192.168.126.18:15443 (reused: true, wasIdle: true, idle: 143.177µs)

    # 2
    $ gurl https://192.168.126.18:15443/ -pso -n2 -tlcp -k
    option TLCP.Version: TLCP
    option TLCP.HandshakeComplete: true
    option TLCP.DidResume: false
    
    Conn-Session: 2.0.1.1:62293->192.168.126.18:15443 (reused: false, wasIdle: false, idle: 0s)
    option TLCP.Version: TLCP
    option TLCP.HandshakeComplete: true
    option TLCP.DidResume: true
    
    Conn-Session: 2.0.1.1:62297->192.168.126.18:15443 (reused: false, wasIdle: false, idle: 0s)
    
    # 3
    $ TLS_SESSION_CACHE=0 gurl https://192.168.126.18:15443/ -pso -n2 -tlcp -k
    option TLCP.Version: TLCP
    option TLCP.HandshakeComplete: true
    option TLCP.DidResume: false
    
    Conn-Session: 2.0.1.1:62401->192.168.126.18:15443 (reused: false, wasIdle: false, idle: 0s)
    option TLCP.Version: TLCP
    option TLCP.HandshakeComplete: true
    option TLCP.DidResume: false
    
    Conn-Session: 2.0.1.1:62402->192.168.126.18:15443 (reused: false, wasIdle: false, idle: 0s)
    ```
   
    ```nginx
    worker_processes  1;
    
    events {
        worker_connections  1024;
    }
    
    http {
        include       mime.types;
        default_type  application/octet-stream;
        keepalive_timeout  65;
    
        server {
            listen       22443 ssl;
        
            # 一行命令生成自签名证书
            # openssl req -x509 -newkey rsa:4096 -nodes -out server.crt -keyout server.key -days 365 -subj "/C=CN/O=krkr/OU=OU/CN=*.d5k.co"
            ssl_certificate server.crt;        # 这里为服务器上server.crt的路径
            ssl_certificate_key server.key;    # 这里为服务器上server.key的路径
            ssl_session_cache shared:SSL:10m;
    
            #ssl_client_certificate ca.crt;    # 双向认证
            #ssl_verify_client on;             # 双向认证
        
        
            #ssl_session_cache builtin:1000 shared:SSL:10m;
            ssl_session_timeout 5m;
            ssl_protocols SSLv2 SSLv3 TLSv1.1 TLSv1.2;
            ssl_ciphers  ALL:!ADH:!EXPORT56:RC4+RSA:+HIGH:+MEDIUM:+LOW:+SSLv2:+EXP;
            ssl_prefer_server_ciphers   on;
        
            default_type            text/plain;
            add_header  "Content-Type" "text/html;charset=utf-8";
            location / {
                return 200 "SSL";
            }
        }
    }
    ```

3. 2022年12月06日 支持 Influx 查询返回表格展示，例如 `gurl :10014/query db==metrics q=='select * from "HB_MSSM-Product-server" where time > now() - 5m order by time desc'  -pb`
4. 2022年04月29日 支持 变量替换，例如 `gurl :5003/@ksuid 'name=@姓名' 'sex=@random(男,女)' 'addr=@地址' 'idcard=@身份证' _hl==echo`
5. 2022年04月06日 支持 stdin 读取多个 JSON 文件，作为请求体调用
    `jq -c '.[]' movies.json | gurl :8080/docs -n0`，目标 [docdb](https://github.com/bingoohuang/docdb)
6. 2022年04月03日 在 content length > 2048 时，自动切换到下载模式
7. 2022年04月02日 修复支持 `:8080/docs q==age:50` 的形式
8. 2022年04月02日 下载文件进度条，使用读取字节计算（读取 gzip 编码并且 Content-Length 给定时，进度条才能个正确显示）, 
    `gurl https://github.com/prust/wikipedia-movie-data/raw/master/movies.json`
