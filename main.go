package main

import (
	"github.com/astaxie/beego"
	_ "project_dailyfresh/models" // 执行init函数完成对数据库的连接与创建
	_ "project_dailyfresh/routers"
)

func main() {
	beego.Run()
}
