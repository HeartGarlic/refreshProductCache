package main

import (
	"bytes"
	"encoding/csv"
	"flag"
	"github.com/gosuri/uiprogress"
	"github.com/panjf2000/ants/v2"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// FailData 定义结果集 chan
var FailData chan map[string]int

// UrlData 定义生产者 chan
var UrlData chan string

// ants 协程控制
var wg sync.WaitGroup

// mainWg 协程控制
var mainWg sync.WaitGroup

// LogFile 日志文件
const LogFile string = "./exec.log"

// Logger 日志句柄
var Logger *log.Logger

// Bar 进度条
var Bar *uiprogress.Bar

// MaxGoroutine 最大协程数量
var MaxGoroutine int

// Params 需要拼接的参数
var Params string

func init() {
	// 声明生产者的 chan
	UrlData = make(chan string, 10)
	// 声明结果集的 chan
	FailData = make(chan map[string]int, 10)
	// 创建日志文件
	logFile, err := os.OpenFile(LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0766)
	if nil != err {
		panic(err)
	}
	Logger = log.New(logFile, "", log.Llongfile)
}

// 刷新产品库页面缓存的程序
// 连接数据库, 批量查询需要刷新的url
// 然后写入文件, url的刷新时间
func main() {
	Logger.Println("==============================================================================================")
	// 数据总条数
	var maxTotal int
	// 要打开的文件
	var FileName string
	// 开启的协程数量
	MaxGoroutine = 10
	// 读取命令行输入
	// flag.IntVar(&maxTotal, "t", 100, "请输入要处理的数据总条数(默认100)")
	flag.StringVar(&FileName, "f", "./all.csv", "请输入要处理的文件(默认./all.csv)")
	flag.IntVar(&MaxGoroutine, "c", 10, "请输入协程数量(并发数)(默认10)")
	flag.StringVar(&Params, "p", "", "数据是否需要拼接参数(?nocache=360checache)")
	flag.Parse()

	maxTotal = CountFileLine(FileName)

	// 引入进度条
	uiprogress.Start()
	Bar = uiprogress.AddBar(maxTotal)
	Bar.AppendCompleted()
	Bar.PrependElapsed()

	startTime := time.Now()
	// 启动生产者
	mainWg.Add(1)
	go GetData(FileName)
	// 启动消费者
	mainWg.Add(1)
	go MakeWork()
	// 最终结果输出
	mainWg.Add(1)
	go FailResult()
	// 等待协程处理结束
	mainWg.Wait()
	// 获取程序结束执行时间
	endTime := time.Now()
	Logger.Printf("[date] %s, Success! Exec Time %s \n", time.Now().String(), endTime.Sub(startTime).String())
}

// FailResult 处理最终结果
func FailResult() {
	// 协程处理结束
	defer mainWg.Done()
	// 读取消费者最终消费结果
	for v := range FailData {
		for key, val := range v {
			Logger.Printf("[dataTime] %s, [Url] %s, [statusCode] %d \n", time.Now().String(), key, val)
		}
		Bar.Incr()
	}
}

// MakeWork 启动消费者
func MakeWork() {
	// 协程执行完成
	defer mainWg.Done()
	// 释放资源
	defer ants.Release()
	// Use the pool with a function,
	// set 10 to the capacity of goroutine pool and 1 second for expired duration.
	// 新建协程池大小为10, 传入方法Exec
	p, _ := ants.NewPoolWithFunc(MaxGoroutine, func(i interface{}) {
		Exec(i)
		wg.Done()
	})
	// 释放协程池资源
	defer p.Release()
	// 循环获取生产者通道中的数据, 并放入到协程池中执行
	for url := range UrlData {
		wg.Add(1)
		_ = p.Invoke(url)
	}
	// 等待所有协程处理完毕
	wg.Wait()
	// 所有协程处理完毕后关闭结果集通道
	close(FailData)
}

// GetData 获取数据, 读取当前定义的 FileName 文件
func GetData(fileName string) {
	//准备读取文件
	fs, err := os.Open(fileName)
	if err != nil {
		Logger.Fatalf("[error] can not open the file, err is %+v", err)
	}
	// 关闭文件
	defer fs.Close()
	// 创建csv文件读取句柄
	r := csv.NewReader(fs)
	//针对大文件，一行一行的读取文件
	for {
		// 循环开始一行一行读取文件
		row, err := r.Read()
		// 如果读取失败 就停止
		if err != nil && err != io.EOF {
			log.Fatalf("[error] can not read, err is %+v", err)
		}
		// 如果读取到文件结尾, 就跳出循环
		if err == io.EOF {
			break
		}
		// 写入文件第一列数据生产者chan
		temp := row[0]
		prefix := "&"
		if Params != "" {
			// 判断url中是否已经存在参数
			if strings.Index(temp, "?") == -1 {
				prefix = "?"
			}
			temp = temp + prefix + Params
		}
		UrlData <- temp
	}
	// 进程执行完成
	defer mainWg.Done()
	// 关闭生产者通道
	defer close(UrlData)
}

// Exec 执行主要逻辑
// 获取url, 请求url地址
// 并返回url请求的状态码
func Exec(url interface{}) {
	resp, err := http.Get(url.(string))
	statusCode := 0
	if err != nil {
		statusCode = 0
	}
	statusCode = resp.StatusCode
	// 写入请求结果到结果集通道
	FailData <- map[string]int{url.(string): statusCode}
	return
}

// CountFileLine 统计当前要打开文件的总行数
func CountFileLine(fileName string) int{
	// 获取文件总行数
	f, err := os.Open(fileName)
	defer f.Close()
	if err != nil {
		log.Fatalln("文件打开失败: ", err)
		return 0
	}
	maxTotal, err := LineCounter(f)
	if err != nil {
		log.Fatalln("文件总行数统计失败: ", err)
		return 0
	}
	return maxTotal
}

// LineCounter count 文件总行数
func LineCounter(r io.Reader) (int, error) {
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count, nil

		case err != nil:
			return count, err
		}
	}
}