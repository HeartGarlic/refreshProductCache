# 清理构建目录
clear:
	rm -rf ./builds
	rm ./exec.log

# 默认构建方法
build:
	go build -o refresh_url main.go
	mkdir builds
	mv refresh_url ./builds

# 构建linux可以执行程序
buildLinux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o refresh_url_linux main.go
	mkdir builds
	mv refresh_url_linux ./builds


