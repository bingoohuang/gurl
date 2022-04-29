# changes

1. 2022年04月29日 支持 变量替换，例如 `gurl :5003/@ksuid 'name=@姓名' 'sex=@random(男,女)' 'addr=@地址' 'idcard=@身份证' _hl==echo`
2. 2022年04月06日 支持 stdin 读取多个 JSON 文件，作为请求体调用
    `jq -c '.[]' movies.json | gurl :8080/docs -n0`，目标 [docdb](https://github.com/bingoohuang/docdb)
3. 2022年04月03日 在 content length > 2048 时，自动切换到下载模式
4. 2022年04月02日 修复支持 `:8080/docs q==age:50` 的形式
5. 2022年04月02日 下载文件进度条，使用读取字节计算（读取 gzip 编码并且 Content-Length 给定时，进度条才能个正确显示）, 
    `gurl https://github.com/prust/wikipedia-movie-data/raw/master/movies.json`
