## 请对你创建的 goroutine 负责

---

### 不要创建一个不知道何时退出的 goroutine
请阅读下面这段代码，看看有什么问题？
```go
package main

import (
	"log"
	"net/http"
)

func setup() {
	// 这里面有一些初始化的操作
}

func main() {
	setup()

	// 主服务
	server()

	// for debug
	pprof()

	select {}
}

func server() {
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/ping", func(writer http.ResponseWriter, request *http.Request) {
			writer.Write([]byte("pong"))
		})

		// 主服务
		if err := http.ListenAndServe(":8080", mux); err != nil {
			log.Panicf("http server err: %+v", err)
			return
		}
	}()
}

func pprof() {
	// 辅助服务，监听了其他端口，这里是 pprof 服务，用于 debug
	go http.ListenAndServe(":8081", nil)
}

```
请问：

- 如果 `server` 是在其它包里面，如果没有特殊说明，如何得知这是一个异步调用呢？
- `main` 函数当中最后在那里空转干什么？会不会存在浪费？
   - `select{}` 会一直阻塞当前 goroutine 直到某个 case 可以执行，因此会存在浪费
- 如果线上出现事故，debug 服务已经退出，这时要如何 debug？
- 如果某一天服务重启，却找不到事故日志，是否能够想起这个 `8081` 端口的服务

### 请将选择权留给别人，不要帮别人做选择
请把是否并发的选择权交给调用者，而不是直接用上了 goroutine
下面这次改动将两个函数是否并发操作的选择权留给了 main 函数
```go
func setup() {
	// 这里面有一些初始化的操作
}

func main() {
	setup()

	// for debug
	go pprof()
    
	// 主服务
	go server()

	select {}
}

func server() {
    mux := http.NewServeMux()
    mux.HandleFunc("/ping", func(writer http.ResponseWriter, request *http.Request) {
        writer.Write([]byte("pong"))
    })

    // 主服务
    if err := http.ListenAndServe(":8080", mux); err != nil {
        log.Panicf("http server err: %+v", err)
        return
    }
}

func pprof() {
	// 辅助服务，监听了其他端口，这里是 pprof 服务，用于 debug
	http.ListenAndServe(":8081", nil)
}

```
### 请不要做一个旁观者
一般情况下，不要让主进程成为一个旁观者，明明可以干活，但是最后使用 `select` 在空跑
```go
package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
)

func setup() {
	// 这里面有一些初始化的操作
}

func main() {
	setup()

	// for debug
	go pprof()

	// 主服务
	server()
}

func server() {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	// 主服务
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Panicf("http server err: %+v", err)
		return
	}
}

func pprof() {
	// 辅助服务，监听了其他端口，这里是 pprof 服务，用于 debug
	http.ListenAndServe(":8081", nil)
}
```
### 不要创建一个你永远不知道什么时候回退出的 goroutine
我们再做一些改造，使用 channel 来控制，解释都写在代码注释里面了
```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

func setup() {
	// 这里面有一些初始化的操作
}

func main() {
	setup()

	// 用于监听服务退出
	done := make(chan error, 2)
	// 用于控制服务退出，传入同一个 stop，做到只要有一个服务退出了那么另外一个服务也会随之退出
	stop := make(chan struct{}, 0)
	// for debug
	go func() {
		done <- pprof(stop)
	}()

	// 主服务
	go func() {
		done <- app(stop)
	}()

	// stopped 用于判断当前 stop 的状态
	var stopped bool
	// 这里循环读取 done 这个 channel
	// 只有有一个退出了，我们就关闭 stop channel
	for i := 0; i < cap(done); i++ {
		if err := <-done; err != nil {
			log.Printf("server exit err: %+v", err)
		}

		if !stopped {
			stopped = true
			close(stop)
		}
	}
}

func app(stop <-chan struct{}) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("pong"))
	})

	return server(mux, ":8080", stop)
}

func pprof(stop <-chan struct{}) error {
	// 注意这里主要是为了模拟服务意外退出，用于验证一个服务退出，其他服务同事退出的场景
	go func() {
		server(http.DefaultServeMux, ":8081", stop)
	}()

	time.Sleep(5 * time.Second)
	return fmt.Errorf("mock pprof exit")
}

func server(handler http.Handler, addr string, stop <-chan struct{}) error {
	s := http.Server{
		Handler: handler,
		Addr:    addr,
	}

	// 这个 goroutine 我们可以控制退出，因为只要 stop 这个 channel close 或者是写入数据，整理就会退出
	// 同时因为调用了 s.Shutdown 之后，http 这个函数启动的 http server 也会优雅退出
	go func() {
		<-stop
		log.Printf("server will exiting, addr: %s", addr)
		s.Shutdown(context.Background())
	}()

	return s.ListenAndServe()
}

```
> 上面的整个流程
> - app() 和 pprof() 都调用了 server() 
> - 5秒后，pprof() 返回了错误: "mock pprof exit" （63-64 line）
> - 此时 done 中接收到了 error （24 line）
> - main 中一直在循环读取 done （36 line）
>    - cap(done) = 2
>    - done 初始为空，因此 <- done 会阻塞，直至 done 中有值
> - **打印 "server exit err: mock pprof exit"（38 line）**
> - 关闭 stop 通道（43 line）
> - server() 中一个 goroutine 监听 stop（76 line）
> - 此时 stop 已关闭，**打印 "server will exiting, addr:" （77 line）**
>    - stop 初始为空 <- stop 会阻塞
>    - stop 被关闭后，从关闭的通道中接收值为 当前数据类型的零值
> - 执行 s.Shutdown，server() 返回 error（78 line）
> - app() 接收到 server() 返回的 error 再返回（54 line）
> - 此时 done 中再次接收到了 error （29 line）
> - main 中一直在循环读取 done （36 line）
> - **打印 "server exit err: http: Server closed"（38 line）**
> - 此时 stopped 已经为 true ，main 退出，程序结束（41 line）

### 不要创建一个永远都无法退出的 goroutine [goroutine 泄露]
再看下面一个例子，复用一下上面的 server 代码
```go
func leak(w http.ResponseWriter, r *http.Request) {
	ch := make(chan bool, 0)
	go func() {
		fmt.Println("异步任务做一些操作")
		<-ch
	}()

	w.Write([]byte("will leak"))
}
```
绝大部分的 goroutine泄露都是因为 goroutine 当中因为各种原因阻塞了，我们在外面也没有控制它退出的方式，所以就泄露了
接下来我们验证一下是不是真的泄露了
启动之后我们访问一下
> 这里需要引入  `"_net/http/pprof"`
> ![image.png](https://cdn.nlark.com/yuque/0/2024/png/29562729/1709882552774-18bff2b7-7292-4575-b763-db96c99af34e.png#averageHue=%23302e2c&clientId=ua4c2e8b1-cd48-4&from=paste&height=163&id=u26fc4c6b&originHeight=163&originWidth=474&originalType=binary&ratio=1&rotation=0&showTitle=false&size=28965&status=done&style=none&taskId=u6bb04f49-f7c1-4935-9214-ad3a721b3c2&title=&width=474)

[http://localhost:8081/debug/pprof/goroutine?debug=1](http://localhost:8081/debug/pprof/goroutine?debug=1) 查看当前的 goroutine 个数为 8
```go
goroutine profile: total 8
2 @ 0xa5bd16 0xa26a5d 0xa26598 0xca517e 0xa8be81
#	0xca517d	main.server.func1+0x3d	D:/Program/project/go/Go-000/Week03/blog/01_goroutine/05/05.go:71
```
然后我们再访问几次 [http://localhost:8080/leak](http://localhost:8080/leak) 可以发现 goroutine 增加到了 75 个，而且一直不会下降
```go
goroutine profile: total 17
7 @ 0xa5bd16 0xa26a5d 0xa26598 0xca5367 0xa8be81
#	0xca5366	main.leak.func1+0x66	D:/Program/project/go/Go-000/Week03/blog/01_goroutine/05/05.go:83

```
### 确保创建出的 goroutine 的工作已经完成
这个其实就是优雅退出的问题，我们可能启动了很多的 goroutine 去处理一些问题，但是服务退出的时候我们并没有考虑到就直接退出了。例如退出前日志还没有 flush 到磁盘，我们的请求还没完全关闭，异步 worker 中还有 job 在执行等等
下面这个例子，假设现在有一个埋点服务，每次请求我们都会上报一些信息到埋点服务上
```go
// Reporter 埋点服务上报
type Reporter struct {
	worker   int
	messages chan string
	wg       sync.WaitGroup
	closed   bool
}

// NewReporter NewReporter
func NewReporter(worker, buffer int) *Reporter {
	return &Reporter{worker: worker, messages: make(chan string, buffer)}
}

func (r *Reporter) run(stop <-chan struct{}) {
	go func() {
		<-stop
		r.shutdown()
	}()

	for i := 0; i < r.worker; i++ {
		r.wg.Add(1)
		go func() {
			for msg := range r.messages {
				time.Sleep(5 * time.Second)
				fmt.Printf("report: %s\n", msg)
			}
			r.wg.Done()
		}()
	}
	r.wg.Wait()
}

func (r *Reporter) shutdown() {
	r.closed = true
	// 注意，这个一定要在主服务结束之后再执行，避免关闭 channel 还有其他地方在啊写入
	close(r.messages)
}

// 模拟耗时
func (r *Reporter) report(data string) {
	if r.closed {
		return
	}
	r.messages <- data
}
```
