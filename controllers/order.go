package controllers

import (
	"fmt"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"github.com/gomodule/redigo/redis"
	"github.com/smartwalle/alipay"
	"project_dailyfresh/models"
	"strconv"
	"strings"
	"time"
)

type GoodsOrderController struct {
	beego.Controller
}

//显示订单页面
func (this *GoodsOrderController) ShowOrder() {
	//	获取数据 -- skuId切片
	skuIds := this.GetStrings("skuId") // 获取多个  -- 只包含选中的商品ID
	// 校验数据
	if len(skuIds) == 0 {
		fmt.Println("前端传递数据出错")
		this.Redirect("/user/cart", 302)
		return
	}
	// 获取用户信息  -- 查询购物车与用户相关,查出对应商品添加购物车时的数量
	username := this.GetSession("username")
	if username == nil {
		fmt.Println("用户未登录")
		this.Redirect("/user/login", 302)
		return
	}
	o := orm.NewOrm()
	var user models.User
	user.UserName = username.(string)
	o.Read(&user, "UserName")
	//连接redis数据库
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		fmt.Println("展示订单出错", err)
		this.Redirect("/user/cart", 302)
		return
	}
	//定义返回数类型变量  -- map切片
	orderGoods := make([]map[string]interface{}, len(skuIds))
	totalPrice := 0 // 总价
	totalCount := 0 // 总数量  -- 显示订单需要，也可以在前端做
	for index, skuId := range skuIds {
		// 将skuId转换为整型
		id, _ := strconv.Atoi(skuId)
		//	1.根据商品ID去查询添加购物车时的数量；2.根据商品ID查询商品详情表，因为要展示图片  --- 需要一个map[string]interface 存起来，然后查完一个商品后放进返回变量orderGoods
		temp := make(map[string]interface{})
		// 根据Id 查购物车情况--得到添加时的数量
		skuCount, _ := redis.Int(conn.Do("hget", "cart_"+strconv.Itoa(user.Id), id))
		// 每件商品添加购物车的数量
		temp["count"] = skuCount
		//存放商品详情变量
		goodsInfo := models.GoodsSKU{
			ID: id,
		}
		//	根据商品Id查询商品详情表
		o.Read(&goodsInfo, "ID")
		//	每件商品的详情存放在temp变量
		temp["goodsSKU"] = goodsInfo
		//	计算每件商品的总价  -- 总数 * 单价
		goodsPrice := goodsInfo.Price * skuCount
		//	每件商品的总价存放temp -- 因为前端会显示。这个也可以在前端做
		temp["goodsPrice"] = goodsPrice
		// 	统计整个商品的订单商品的总数 ，总价
		totalPrice += goodsPrice
		totalCount += skuCount
		// 将每一件商品的信息 temp 存放在切片  orderGoods中
		orderGoods[index] = temp
	}
	//	将订单商品信息传送
	this.Data["orderGoods"] = orderGoods
	this.Data["totalCount"] = totalCount
	this.Data["totalPrice"] = totalPrice
	//	传递用户信息
	GetUser(&this.Controller)
	//获取用户收获地址 -- 一个用户有多个地址
	var addrs []models.Address
	o.QueryTable("Address").RelatedSel("User").Filter("User__Id", user.Id).All(&addrs)
	//	传递用户地址
	this.Data["addrs"] = addrs
	//	传递购物车的商品ID切片
	this.Data["skuIds"] = skuIds
	transferPrice := 10
	this.Data["transferPrice"] = transferPrice
	// 购物车总价 ：包含运费
	this.Data["realyPrice"] = totalPrice + transferPrice
	this.TplName = "place_order.html"
}

//添加订单
func (this *GoodsOrderController) HandleAddOrder() {
	//	获取数据
	addrid, _ := this.GetInt("addrid")
	payId, _ := this.GetInt("payId")
	getskuids := this.GetString("skuids")
	ids := getskuids[1 : len(getskuids)-1]
	skuids := strings.Split(ids, " ")
	transferPrice, _ := this.GetInt("transferPrice") // 运费
	realyPrice, _ := this.GetInt("realyPrice")
	totalCount, _ := this.GetInt("totalCount")
	//fmt.Println(addrid, payId, skuids, totalCount, transferPrice, realyPrice)
	//	定义返回数据类型 -- Ajax返回数据为json格式，在beego中使用map
	var resp = make(map[string]interface{})
	defer this.ServeJSON() // 发送json数据必须要做的一步，因为下面会多次返回数据因此在函数结束时统一调用，这样就不用每次发送时都调用
	// 校验数据  -- 这里只做简单的校验
	if len(skuids) == 0 {
		resp["code"] = 400 //前端传递数据出错，转态码
		resp["msg"] = "数据传递出错"
		this.Data["json"] = resp
		return
	}
	//	向订单表中插入数据
	o := orm.NewOrm()
	o.Begin() // 标识事物开始 这里要使用到事物，因为订单在添加时可能出现各种出错，比如库存不足，这时就应该不插入数据库，所有的操作都需要回滚，要么一起提交成功，要么一个出错都会回滚
	username := this.GetSession("username")
	if username == nil {
		resp["code"] = 300 //用户未登录
		resp["msg"] = "用户未登录"
		this.Data["json"] = resp
		return
	}
	//	获取用户信息
	user := models.User{UserName: username.(string)}
	o.Read(&user, "UserName")
	//	定义订单详情表的初始化变量
	var order models.OrderInfo
	//	为了防止订单号重复，使用当前时间加用户Id，就算同一时间可能有多个事件提交，但是同一时间类也能有一个用户提交 -- 保证唯一性
	order.OrderId = time.Now().Format("2006010215030405") + strconv.Itoa(user.Id)
	order.User = &user
	order.OrderStatus = 1 //支付状态
	order.PayMethod = payId
	order.TotalCount = totalCount
	order.TotalPrice = realyPrice
	order.TransitPrice = transferPrice
	// 获取用户的地址信息, 更具传递过来的地址ID--用户添加订单选择的地址
	var addr models.Address
	addr.Id = addrid
	o.Read(&addr)
	order.Address = &addr
	// 执行插入操作 -- 完成订单信息表的插入
	o.Insert(&order)
	conn, err := redis.Dial("tcp", ":6379")
	defer conn.Close()
	//	循环添加订单传过来的商品Id切片，在根据每一个Id获取商品详情，
	for _, skuId := range skuids {
		//根据商品ID获取商品详情
		id, _ := strconv.Atoi(skuId)
		//	商品详情类型变量
		var goods models.GoodsSKU
		goods.ID = id
		waitTime := 3 // 循环比较订单插入数据库时，库存的变化
		for waitTime > 0 {
			o.Read(&goods)
			//获取商品详情表中的库存
			preStock := goods.Stock
			//	订单商品信息表的插入 --这里需要当前商品id添加购物车的数量，要进行商品详情表中的库存进行比较，如果库存不够则回滚前面操作，就算前面已经添加了订单表也会删除，因为添加了事物
			var orderGoods models.OrderGoods
			// 订单商品表存放对应订单的详细信息
			orderGoods.GoodsSKU = &goods
			//	订单信息表存放订单信息
			orderGoods.OrderInfo = &order
			//	获取添加购物车时对应商品的添加（想要购买）数量---从redis中拿
			if err != nil {
				fmt.Println("添加订单报错，redis连接失败", err)
				resp["code"] = 500 //服务器内部错误
				resp["msg"] = "服务端内部错误"
				this.Data["json"] = resp
				return
			}
			//	获取对应用户商品添加购物车时的数量
			buyCount, err := redis.Int(conn.Do("hget", "cart_"+strconv.Itoa(user.Id), id))
			// 根据商品详情表中的库存进行对比，若库存不够，则要进行回滚，前面已经插入成功的order表也要删除 -- 事物的作用
			if err != nil {
				fmt.Println("添加订单报错：查询购物车错误", err)
				resp["code"] = 500 //库存不足状态码
				resp["msg"] = "服务器内部错误"
				this.Data["json"] = resp
				o.Rollback() // 回滚
				return
			}
			if buyCount > goods.Stock {
				resp["code"] = 300 //库存不足状态码
				resp["msg"] = "库存不足，购买失败"
				this.Data["json"] = resp
				o.Rollback() // 回滚
				return
			}
			orderGoods.Count = buyCount
			orderGoods.Price = buyCount * goods.Price // 订单每一件商品的总价
			o.Insert(&orderGoods)                     // 将每一件商品的订单商品信息表插入数据库
			// 更新添加订单成功的库存，删除购物车数据
			goods.Stock -= buyCount
			// 销量增加
			goods.Sale += buyCount
			//	更新商品详情表 -- 更新时根据商品ID和商品以前库存，如果在购买的时候库存发生变化，就会更新不成功（可能由于并发其他人也在你购买时下单成功，库存发生变化，因此防止库存不够，因此要阻止每次购买，进行循环重新判断购买量和库存在进行购买）
			update, _ := o.QueryTable("GoodsSKU").Filter("ID", goods.ID).Filter("Stock", preStock).Update(orm.Params{"Stock": goods.Stock, "Sale": goods.Sale})
			if update == 0 {
				//更新失败
				if waitTime > 0 {
					waitTime -= 1
					continue
				}
				resp["code"] = 300
				resp["msg"] = "添加订单前，订单库存改变，添加失败"
				this.Data["json"] = resp
				o.Rollback()
				return
			} else {
				//删除购物车
				conn.Do("hdel", "cart_"+strconv.Itoa(user.Id), goods.ID)
				break
			}
		}
	}
	//	传递数据
	o.Commit()         // 执行到这里，代码没有出错，说明 可以提交事物，上面的订单表order，订单商品信息表orderGoods 才会被真正的插入数据库
	resp["code"] = 200 // 处理业务成功状态码
	resp["msg"] = "添加订单成功"
	this.Data["json"] = resp
}

//支付业务
func (this *GoodsOrderController) HandlePay() {
	var alipublickkey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv8d9kmjASjhHebTECOGd0sQD8sFQr1WfPxCLLNbuvehQTdgjNGlTYvOMlAYHNwGNvnXg35ZIpbNY0ynUvtM/+6Hk11oqh4fvmcYPFX4BqH0h7EBqiO3LYL3Rkx/Ui/ICHxt4K+PT1MISKwvaYccgbtlHKKyHVUDy5AZHncwSf+fUKgrKLUx12FUueqW0ZtvXEVKPockHAv1Ht55iM8VuxBnLFXJMZ0UmdWzXeoMbdMC+5ED3SS6VHJ+bF6PBHsfHT3OVF4O0pIy0PrdABrPsyHL14XsuokVj6+fRyTrvxKVETF1f1NOVxxXcq6VPdSNSR6jlAFZNwSmi7HKEUPN1KwIDAQAB"
	var privatekey = "MIIEpQIBAAKCAQEAv8d9kmjASjhHebTECOGd0sQD8sFQr1WfPxCLLNbuvehQTdgjNGlTYvOMlAYHNwGNvnXg35ZIpbNY0ynUvtM/+6Hk11oqh4fvmcYPFX4BqH0h7EBqiO3LYL3Rkx/Ui/ICHxt4K+PT1MISKwvaYccgbtlHKKyHVUDy5AZHncwSf+fUKgrKLUx12FUueqW0ZtvXEVKPockHAv1Ht55iM8VuxBnLFXJMZ0UmdWzXeoMbdMC+5ED3SS6VHJ+bF6PBHsfHT3OVF4O0pIy0PrdABrPsyHL14XsuokVj6+fRyTrvxKVETF1f1NOVxxXcq6VPdSNSR6jlAFZNwSmi7HKEUPN1KwIDAQABAoIBAE9NUqOkJT+LniK5mQaDJRvuaiOLxK18HmmZkbNs/TQSSIKoCYa2twCH7W2YQIuCXPaRD/fk0Q2T5/sJpStzd1W6UEKsykFY+L8Bo2Mjw9PESq7CxEry6dKLK4pG80EbRb1PQpYDk6i6x4B9WkRsbwDnYAF1tlCOluGrpxmdNVkl7z3Ts2c0VP2zpD7h05IbxSvcweQmbjSvyXEkhxzbS01ZwhW10cBiRhbE/fbbBo7MioBrs21lHXGhkwtOHsV9pU0vqcDqrKuI5+sN\nLTEgUJV1ZBBZJWKCFLaKWASH53hhGfW637ubkipElpJdq858Lh6qXj8CBdOEZdB54kkwHeECgYEA/PXh25IwdsLpnwm+FpY+gIPst0TnfGNNMYpXow88sTODhVUyvYU0fs9tfFPh+TlASU6tcsUZj6bklk9jyZWMEXtJOF6x4NxgVNK8MxVMjZ5Id1AppIyRFArlZIUzLduC2coWgCmTLOQvFtiuIHajOXLwuntA+M8Bj8eqQhPTRzcCgYEAwhVpfyXJUaB3ccsBNlHZT63DH0dNXETLMmTqnzz/KQWjUF+0vM9cWsgqKR+q3oCySWdEgjNdu4ZKQkob+Z4zVUvwGzISH7+oFtXYvRRXcpuUske833v+/AanOZ95HYn749nI\nAyoRYyAgY3B0iXPJxiDagUNyULMIb3JSan1W060CgYEA2QCiAbe2ZZstySYVcNDwy1ThFBNDNh0F0rLoHVTr7uPPNulwvs5vyz1sohRfrWoksP6SovtcwzQbsqpmYz8sSq7lkDsEA29qIDosAvKJmo+ngNs+7g88QeJbCVGPJw7BgM3xYX7I5+DUWJgHQIgl3BmzU3Z6tTb4EvzpHQhe3h8CgYEAsWzxrJLWoBCaIST8TrQ0fWrUXdvJFPiu6brn4frZKJ9G1Uso5xKJ01P5du7EPfRZCFGnh399yNjTOhaVzHSbaPaq4bG8b9m9yGJmaTQXXWZtYS3DtGqeh7dtWHg5OI/T/lAxUPM8Qeo0sbM0VhPL+Zw/JLyL3MpOg9N3FHLQ1WECgYEA6Mrvl0nheAOdvr2yhyDn7h3rMch0Wi5UANcmX/NuNX5+H2Yr5DRmWjCvyqzZUOfeCKbNNjxshZb2BHdVtDjVVVn2UoRfBq7YnH54pdQA2W0R3xqckd28kZjv5alOIBBqRO1JEnrKLylbsJpb5miLrkjgEroSbN8+xpGIUPaviAU="
	var appId = "2021000119656181"
	var client = alipay.New(appId, alipublickkey, privatekey, false)
	//获取数据
	orderId := this.GetString("orderId")
	totalPrice := this.GetString("totalPrice")
	var p = alipay.AliPayTradePagePay{}
	p.NotifyURL = "http://xxx"
	p.ReturnURL = "http://192.168.248.1:8080/user/payok"
	p.Subject = "天天生鲜购物平台"
	p.OutTradeNo = orderId
	p.TotalAmount = totalPrice
	p.ProductCode = "FAST_INSTANT_TRADE_PAY"
	var url, err = client.TradePagePay(p)
	if err != nil {
		fmt.Print("支付失败", err)
		return
	}
	var payURL = url.String()
	this.Redirect(payURL, 302)
}

//支付成功业务
func (this *GoodsOrderController) HandlePayOk() {
	//	获取数据
	orderId := this.GetString("out_trade_no")
	//	校验数据
	if orderId == "" {
		fmt.Println("支付返回数据错误")
		this.Redirect("/user/userOrder", 302)
	}
	//	更新支付状态
	o := orm.NewOrm()
	//	 根据返回的字符订单Id将订单信息表中的支付状态进行更改
	count, err := o.QueryTable("OrderInfo").Filter("OrderId", orderId).Update(orm.Params{"OrderStatus": 2})
	if err != nil {
		fmt.Println("更新支付状态失败")
		this.Redirect("/user/userOrder", 302)
		return
	}
	if count == 0 {
		fmt.Println("更新数据失败")
		this.Redirect("/user/userOrder", 302)
		return
	}
	//	返回视图
	this.Redirect("/user/userOrder", 302)
}
