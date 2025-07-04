
# GoAdmin 介绍

GoAdmin 是一个帮你快速搭建数据可视化管理应用平台的框架。 

- [github](https://github.com/GoAdminGroup/go-admin)
- [论坛](http://discuss.go-admin.com)
- [文档](https://book.go-admin.cn)

## 目录介绍

```
.
├── Makefile            Makefile
├── adm.ini             adm配置文件
├── build               二进制构建目标文件夹
├── config.yml          配置文件
├── go.mod              go.mod
├── go.sum              go.sum
├── html                前端html文件
├── logs                日志
├── main.go             入口文件
├── main_test.go        测试文件
├── pages               页面控制器
├── tables              数据模型
└── uploads             上传文件夹
```

## 生成CRUD数据模型

### 在线工具

管理员身份运行后，访问：http://127.0.0.1:9094/admin/info/generate/new

### 使用命令行工具

```
adm generate -l cn -c adm.ini
```

