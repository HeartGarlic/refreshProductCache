# golang 批量请求url代码

### 依赖检测
go mod tidy

### 下载依赖包
go mod download

### 导出依赖包
go mod vendor

### 以下命令会自动导出依赖
go build -o refreshUrl main.go

#### 执行 -c 并发数 -f 数据文件路径 -p url 中需要拼接的参数
./refreshUrl -c 10 -f ./all.csv -p "nocache=360checache"

#### 定时任务设置
0 3 * * * /path/to/refreshUrl -c 10 -f ./all.csv -p "nocache=360checache"