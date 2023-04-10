# changes

1. 2023年04月10日 支持 TLS SESSION REUSE

    ```shell
    $ gurl https://192.168.126.18:22443/ -pso -n2 -k
    Conn-Session: 2.0.1.1:53026->192.168.126.18:22443 (reused: false, wasIdle: false, idle: 0s)
    option TLS.Version: TLSv12
    option TLS.HandshakeComplete: true
    option TLS.DidResume: false
    
    Conn-Session: 2.0.1.1:53027->192.168.126.18:22443 (reused: false, wasIdle: false, idle: 0s)
    option TLS.Version: TLSv12
    option TLS.HandshakeComplete: true
    option TLS.DidResume: true
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

2. 2022年12月06日 支持 Influx 查询返回表格展示，例如 `gurl :10014/query db==metrics q=='select * from "HB_MSSM-Product-server" where time > now() - 5m order by time desc'  -pb`
3. 2022年04月29日 支持 变量替换，例如 `gurl :5003/@ksuid 'name=@姓名' 'sex=@random(男,女)' 'addr=@地址' 'idcard=@身份证' _hl==echo`
4. 2022年04月06日 支持 stdin 读取多个 JSON 文件，作为请求体调用
    `jq -c '.[]' movies.json | gurl :8080/docs -n0`，目标 [docdb](https://github.com/bingoohuang/docdb)
5. 2022年04月03日 在 content length > 2048 时，自动切换到下载模式
6. 2022年04月02日 修复支持 `:8080/docs q==age:50` 的形式
7. 2022年04月02日 下载文件进度条，使用读取字节计算（读取 gzip 编码并且 Content-Length 给定时，进度条才能个正确显示）, 
    `gurl https://github.com/prust/wikipedia-movie-data/raw/master/movies.json`
