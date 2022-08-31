# internal

## resuming download

[How can I find out whether a server supports the Range header?](https://stackoverflow.com/questions/720419/how-can-i-find-out-whether-a-server-supports-the-range-header)

The way the [HTTP spec](http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.35.2) defines it, if the server knows how to support the Range header, it will. That in turn, requires it to return a [206 Partial Content](http://www.w3.org/Protocols/rfc2616/rfc2616-sec10.html#sec10.2.7) response code with a [Content-Range](http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.16) header, when it returns content to you. Otherwise, it will simply ignore the Range header in your request, and return a 200 response code.

This might seem silly, but are you sure you're crafting a valid HTTP request header? All too commonly, I forget to specify HTTP/1.1 in the request, or forget to specify the Range specifier, such as "bytes".

Oh, and if all you want to do is check, then just send a HEAD request instead of a GET request. Same headers, same everything, just "HEAD" instead of "GET". If you receive a 206 response, you'll know Range is supported, and otherwise you'll get a 200 response.

This is for others searching how to do this. You can use curl:

`curl -I http://exampleserver.com/example_video.mp4`

In the header you should see

`Accept-Ranges: bytes`

You can go further and test retrieving a range

`curl --header "Range: bytes=100-107" -I http://exampleserver.com/example_vide0.mp4`

and in the headers you should see

```sh
HTTP/1.1 206 Partial Content
Content-Range: bytes 100-107/10000000
Content-Length: 8
```

instead of 10000000 you'll see the length of the file

请求下载整个文件：

```sh
GET /test.rar HTTP/1.1
Connection: close
Host: 116.1.219.219
Range: bytes=0-801 //一般请求下载整个文件是bytes=0- 或不用这个头
```

一般正常回应：

```sh
HTTP/1.1 200 OK
Content-Length: 801
Content-Type: application/octet-stream
Content-Range: bytes 0-800/801 //801:文件总大小
```

版权声明：本文为CSDN博主「king_weng」的原创文章，遵循CC 4.0 BY-SA版权协议，转载请附上原文出处链接及本声明。
[原文链接](https://blog.csdn.net/King_weng/article/details/105691553)

[图解：HTTP 范围请求，助力断点续传、多线程下载的核心原理](https://juejin.cn/post/6844903642034765837)

例如已经下载了 1000 bytes 的资源内容，想接着继续下载之后的资源内容，只要在 HTTP 请求头部，增加 Range: bytes=1000- 就可以了。
Range 还有几种不同的方式来限定范围，可以根据需要灵活定制：

1. 500-1000：指定开始和结束的范围，一般用于多线程下载。
2. 500- ：指定开始区间，一直传递到结束。这个就比较适用于断点续传、或者在线播放等等。
3. -500：无开始区间，只意思是需要最后 500 bytes 的内容实体。
4. 100-300,1000-3000：指定多个范围，这种方式使用的场景很少，了解一下就好了。
