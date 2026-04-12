# FastGin快速开发脚手架

## 项目简介

为了快速开发新项目，每次都需要去做一些相同的操作，例如读取配置文件，写路由，连接gorm，这样很繁琐

所以本项目做好这些事情，只需要在此基础上添砖加瓦即可

## 功能特性

1. 配置文件的读取
2. zap日志
3. gorm连接mysql
4. 命令行参数绑定
5. 内置swagger的api文档
6. 中间件操作-支持认证和限流
7. 通用列表分页查询
8. 密码认证
9. 图片验证码

## 项目运行

```shell
# 安装环境
go mod tidy

# 
go run main.go
```

## 目录说明

```text
cmd/gen          生成query查询
handlers         路由对应的处理器函数
config           配置的struct目录
core             初始化操作
flags            命令行参数
global           全局变量
middleware       gin的中间件
models           表结构
routers          路由
service          服务(redis,定时器cron)
utils            工具目录
main.go          入口文件
settings.yaml    配置文件
```