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
fast-gin/
├── cmd/gen/           # 代码生成工具
├── config/            # 配置文件结构定义
├── core/              # 核心初始化（日志、数据库、Redis）
├── dal/query/         # GORM Gen生成的查询代码
├── flags/             # 命令行参数解析
├── global/            # 全局变量
├── handlers/          # 业务处理层（controller）
│   ├── captcha/       # 验证码
│   ├── image/         # 图片上传
│   ├── rbac/          # 权限管理
│   └── user/          # 用户管理（已有登录注册）
├── middleware/        # 中间件（认证、权限、限流）
├── models/            # 数据模型
├── permissions/       # 权限定义
├── routers/           # 路由定义
├── service/           # 业务服务层
├── utils/             # 工具包
└── settings.yaml      # 主配置文件
```