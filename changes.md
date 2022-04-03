# changes

1. 2022年04月03日 在 content length > 2048 时，自动切换到下载模式
2. 2022年04月02日 修复支持 `:8080/docs q==age:50` 的形式
3. 2022年04月02日 下载文件进度条，使用读取字节计算（读取 zip 编码并且 Content-Length 给定时，进度条才能个正确显示）, `gurl https://github.com/prust/wikipedia-movie-data/raw/master/movies.json -d`