package controllers

import (
	"fmt"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"github.com/gomodule/redigo/redis"
	"project_dailyfresh/models"
	"strconv"
)

type GoodsCartController struct {
	beego.Controller
}

//获取购物车数量函数
func GetCartCount(this *beego.Controller) int {
	//	从redis中获取购物车数量
	//1.获取用户信息， 购物车数量根据用户信息存储
	username := this.GetSession("username")
	if username == nil {
		return 0 // 用户未登录默认显示购物车数量为0
	}
	o := orm.NewOrm()
	user := models.User{UserName: username.(string)}
	err := o.Read(&user, "userName")
	if err != nil {
		fmt.Println("获取用户购物车数量函数出错：根据用户名查询用户失败", err)
		return 0
	}
	//	连接数据库
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		fmt.Print("获取用户购物车数量函数出错:了解redis数据库失败", err)
		return 0
	}
	defer conn.Close() //及时关闭连接
	cartCount, _ := redis.Int(conn.Do("hlen", "cart_"+strconv.Itoa(user.Id)))
	return cartCount
}

//添加购物车业务
func (this *GoodsCartController) HandleAddCart() {
	//	获取Ajax传递的数据
	goodsId, err1 := this.GetInt("skuid")
	count, err2 := this.GetInt("count")
	//向Ajax返回数据，Ajax支持使用json的格式返回数据，在beego中使用map
	resp := make(map[string]interface{})
	defer this.ServeJSON() // 每次在发送时都要调用这个方法，因此在函数关闭时统一调用
	if err2 != nil || err1 != nil {
		fmt.Println("添加购物车出错：获取参数错误")
		// 封装返回信息 -- 向前端提示错误
		resp["code"] = 400 // 自己随便定义的错误状态码
		resp["msg"] = "传递的数据不正确"
		this.Data["json"] = resp
		return
	}
	//虽然请求路径使用了/user/addCart 但是此时由于使用的Ajax，beego的过滤会直接返回html不会展示，即不会重定向到登录页面，因此这里要自己写
	userName := this.GetSession("username") //返回值为interfa类型
	if userName == nil {
		resp["code"] = 300 // 表示用户未登录
		resp["mag"] = "用户未登录"
		this.Data["json"] = resp
		return
	}
	// 用户登录转态  -- 根据用户信息，将商品ID和数量存入redis数据库,使用redis哈希的数据结构
	o := orm.NewOrm()
	user := models.User{UserName: userName.(string)} //使用类型断言的方式将其转换为string
	o.Read(&user, "UserName")
	//连接数据库
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		fmt.Println("添加购物车：连接数据库出错:", err)
		resp["code"] = 500 // 服务器内部出错
		resp["mag"] = "服务器内部出错"
		this.Data["json"] = resp
		return
	}
	//先获取原来的数量，然后给数量加起来 -- 不然这次新查到的数量为把以前的覆盖，我们需要进行累加
	preCount, err1 := redis.Int(conn.Do("hget", "cart_"+strconv.Itoa(user.Id), goodsId))
	if err1 != nil { //说明第一次添加
		conn.Do("hset", "cart_"+strconv.Itoa(user.Id), goodsId, count)
	} else {
		//已经添加过，存储有上一次添加
		_, err2 = conn.Do("hset", "cart_"+strconv.Itoa(user.Id), goodsId, count+preCount)
		if err2 != nil {
			fmt.Println("添加购物车：向redis存储数据出错:", err)
			resp["code"] = 500 // 服务器内部出错
			resp["mag"] = "服务器内部出错"
			this.Data["json"] = resp
		}
	}

	rep, err := conn.Do("hlen", "cart_"+strconv.Itoa(user.Id)) // 获取数据库的数据数量 -- 对应购物车数量，购物车有几件商品
	cartCount, _ := redis.Int(rep, err)                        // 回复助手函数， redis查询到的数据为interface类型，使用对应的回复助手函数转换为对应类型
	//将数据返回
	resp["code"] = 200 //成功转态码
	resp["msg"] = "添加购物车成功"
	resp["cartCount"] = cartCount
	this.Data["json"] = resp
}

//显示购物车页面
func (this *GoodsCartController) ShowCart() {
	GetUser(&this.Controller)
	//获取用户信息
	username := this.GetSession("username") // 返回类型为interface，需要使用类型断言进行转换为相应的类型
	if username == nil {
		fmt.Println("用户未登录") // 正常情况，未登录用户进入不了购物车展示页面，非法进入
		this.Redirect("/login", 302)
		return
	}
	o := orm.NewOrm()
	var user models.User
	user.UserName = username.(string)
	o.Read(&user, "userName")
	// 查询redis数据库，获取到登录用户购物车信息
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		fmt.Println("显示购物车页面报错：连接redis出错", err)
	}
	goodsMap, _ := redis.IntMap(conn.Do("hgetall", "cart_"+strconv.Itoa(user.Id))) //返回值为 map[string] int   -- 获取哈希存储的 的所有feild值
	goods := make([]map[string]interface{}, len(goodsMap))
	// 定义一个大容器，里面会存放购物车商品详情，总价，总数量等等
	// 购物车的商品件数 -- 有多少件不同种类的商品 -- 就是len(goodsMap),多少个商品ID就有多少件不同的商品
	cartGoodsCount := len(goodsMap)
	this.Data["cartGoodsCount"] = cartGoodsCount
	i := 0          // 小标  -- 因为在goodsMap range循环中index 不是下标，而是商品ID
	totalPrice := 0 // 用户购物车总价
	totalCount := 0 // 用户购物车总件数
	for index, value := range goodsMap {
		skuId, _ := strconv.Atoi(index) //将商品Id转换为整型
		var goodsSKU models.GoodsSKU    // 存放商品详情容器
		goodsSKU.ID = skuId
		//根据商品ID查询商品详情数据库
		o.Read(&goodsSKU, "ID")
		//	定义与大容器类型相同的map，大容器里面的每一个小容器,上面是切片
		good := make(map[string]interface{})
		good["goodsSKU"] = goodsSKU                //存放每件商品的详情
		good["count"] = value                      //存放每件商品的数量
		good["goodPrice"] = goodsSKU.Price * value //每一件商品的总价：数量 * 单价
		totalPrice += goodsSKU.Price * value       // 这件商品count/value 件，求总价,然后每次循环将每件商品的总价累加，求出整个用户购物车的总价，--因为页面展示需要，也可以通过js做
		totalCount += value
		//	将good添加到 大容器goods中
		goods[i] = good
		i += 1
	}
	this.Data["totalPrice"] = totalPrice
	this.Data["totalCount"] = totalCount
	this.Data["goods"] = goods
	this.TplName = "cart.html"
}

//更新购物车数量
func (this *GoodsCartController) HandleUpdateCartCount() {
	//向Ajax返回数据，Ajax支持使用json的格式返回数据，在beego中使用map
	resp := make(map[string]interface{})
	// beego中每次在发送json数据时都要调用这个方法，因此在函数关闭时统一调用
	defer this.ServeJSON()
	//获取传递的数据
	skuId, err1 := this.GetInt("skuId")
	count, err2 := this.GetInt("count") // 防止传递非法的数据
	if err2 != nil || err1 != nil {
		resp["code"] = 300 // 用户未登录状态码
		resp["msg"] = "前端传递的数据非法"
		this.Data["json"] = resp
		return
	}

	//	获取登录用户信息  -- 更新购物车信息与用户有关
	username := this.GetSession("username")
	if username == nil {
		resp["code"] = 400 // 用户未登录状态码
		resp["msg"] = "用户未登录"
		this.Data["json"] = resp
		return
	}
	// 根据用户名查询用户信息
	o := orm.NewOrm()
	user := models.User{UserName: username.(string)}
	o.Read(&user, "UserName")
	// 根据用户ID，商品ID , 查询redis数据库，并修改商品数量
	conn, err := redis.Dial("tcp", ":6379")
	defer conn.Close() //函数结束关闭连接
	if err != nil {
		fmt.Println("修改购物车报错：连接购物车错误", err)
		resp["code"] = 500 //服务器内部错误状态码
		resp["msg"] = "服务器内部发生错误"
		this.Data["json"] = resp
		return
	}
	//查询要修改的数据
	_, err = conn.Do("hget", "cart_"+strconv.Itoa(user.Id), skuId)
	if err != nil {
		fmt.Println("修改购物车报错：未查询到要更新的购物车商品信息", err)
		resp["code"] = 300 //服务器内部错误状态码
		resp["msg"] = "未查询到要更新的购物车商品信息"
		this.Data["json"] = resp
		return
	}
	//更新购物车商品数据量
	_, err = conn.Do("hset", "cart_"+strconv.Itoa(user.Id), skuId, count)

	if err != nil {
		fmt.Println("修改购物车报错：更新的购物车商品信息出错", err)
		resp["code"] = 500 //服务器内部错误状态码
		resp["msg"] = "更新的购物车商品信息出错"
		this.Data["json"] = resp
		return
	}

	//将最新的购物车商品数量返回页面
	newCount, _ := redis.Int(conn.Do("hget", "cart_"+strconv.Itoa(user.Id), skuId))
	resp["code"] = 200 // 处理业务成功转态码
	resp["msg"] = "修改成功"
	resp["newCount"] = newCount
	this.Data["json"] = resp
}

//根据商品Id删除购物车数据
func (this *GoodsCartController) HandleDelCartGoods() {
	//准备返回数据 -- json在beego中使用 map
	resp := make(map[string]interface{})
	defer this.ServeJSON() // 发送json数据都需要执行
	//获取数据
	skuId, err := this.GetInt("skuId")
	if err != nil {
		resp["code"] = 400 // 参数传递出错转态码
		resp["msg"] = "前端传递参数出错"
		this.Data["json"] = resp
		return
	}
	// 获取用户信息 -- 购物车存储的信息与用户关联
	username := this.GetSession("username")
	if username == nil {
		//	用户未登录 - 一般没有登录的用户进不来这里
		resp["code"] = 300 // 用户未登录转态码
		resp["msg"] = "用户未登录，不能删除"
		this.Data["json"] = resp
		return
	}
	// 根据用户名查询用户完整信息
	o := orm.NewOrm()
	var user models.User
	user.UserName = username.(string)
	o.Read(&user, "UserName")
	//连接数据库
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		resp["code"] = 500 // 服务器内部错误
		resp["msg"] = "服务器内部错误"
		this.Data["json"] = resp
		return
	}
	_, err = conn.Do("hdel", "cart_"+strconv.Itoa(user.Id), skuId)
	if err != nil {
		resp["code"] = 500 // 服务器内部错误
		resp["msg"] = "服务器内部错误,删除失败"
		this.Data["json"] = resp
		return
	}
	// 返回数据
	resp["code"] = 200 // 删除成功转态码
	resp["msg"] = "删除成功"
	this.Data["json"] = resp
}
