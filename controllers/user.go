package controllers

import (
	"encoding/base64"
	"fmt"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"github.com/astaxie/beego/utils"
	"github.com/gomodule/redigo/redis"
	"project_dailyfresh/models"
	"regexp"
	"strconv"
)

type UserController struct {
	beego.Controller
}

//显示注册页面
func (this *UserController) ShowRegister() {
	this.TplName = "register.html"
}

//注册业务
func (this *UserController) HandleRegister() {
	//获取数据
	userName := this.GetString("user_name")
	password := this.GetString("pwd")
	cpassword := this.GetString("cpwd")
	email := this.GetString("email")
	allow := this.GetString("allow")
	fmt.Println(userName, password, cpassword, email, allow)
	//校验数据
	if userName == "" || password == "" || cpassword == "" || email == "" || allow == "" {
		fmt.Println("注册时：注册的相关信息不能为空")
		//向前端展示错误信息
		this.Data["errmsg"] = "数据不完整，请重新注册～"
		this.TplName = "register.html"
		return
	}
	if password != cpassword {
		fmt.Println("注册时：两次密码不一致")
		this.Data["errmsg"] = "两次密码不一致，请重新输入"
		this.TplName = "register.html"
		return
	}
	reg, _ := regexp.Compile("^[A-Za-z0-9\u4e00-\u9fa5]+@[a-zA-Z0-9_-]+(\\.[a-zA-Z0-9_-]+)+$")
	res := reg.FindString(email) //匹配成功返回比配后的字符串，不成功返回空
	if res == "" {
		fmt.Println("注册时：邮箱格式不正确，请重新输入")
		this.Data["errmsg"] = "邮箱格式不正确，请重新输入"
		this.TplName = "register.html"
		return
	}
	//	处理数据
	o := orm.NewOrm()
	var user models.User // 操作的数据对象
	user.UserName = userName
	user.PassWord = password
	user.Email = email
	//插入前先根据用户名检查，注册用户是否已经被注册
	err := o.Read(&user, "userName")
	if err == nil {
		//查到数据，用户名已被注册不能注册
		fmt.Println("注册时：用户名已被注册，请重新输入")
		this.Data["errmsg"] = "用户名已被注册，请重新输入"
		this.TplName = "register.html"
		return
	}
	_, err = o.Insert(&user)
	if err != nil {
		fmt.Println("注册时：插入数据库失败")
		this.Data["errmsg"] = "注册失败,请更换数据注册"
		this.TplName = "register.html"
		return
	}
	//	发送邮件，激活账户
	emailConfig := `{"username": "811191051@qq.com", "password":"uestigkkqqnsbfeg", "host": "smtp.qq.com","port":587}`
	emailConn := utils.NewEMail(emailConfig)
	emailConn.From = "811191051@qq.com" // 从那个邮箱发邮件
	emailConn.To = []string{email}      // 发给谁：   ---可以发多封邮箱  所以是数组类型
	emailConn.Subject = "天天生鲜平台注册邮箱"
	//注意这里我们发送给用户的是激活请求地址
	emailConn.Text = "激活注册账户" // 将整型转换为string
	//url_str := "192.168.248.1/active?Id=" + id
	emailConn.Text = "localhost:8080/active?Id=" + strconv.Itoa(user.Id)
	//emailConn.HTML = "<a href=" + url_str + ">激活账户</a>"
	emailConn.Send() //发送邮件
	this.Ctx.WriteString(`<h1>注册成功，请去注册邮箱进行账户激活</h1><a href='www.baidu.com'>点击跳转</a>`)
}

//激活业务
func (this *UserController) HandleActive() {
	userId, err := this.GetInt("Id")
	if err != nil {
		fmt.Println("激活业务：获取传递的用户Id失败")
		this.Data["errmsg"] = "要激活的用户不存在"
		this.TplName = "register.html"
		return
	}
	o := orm.NewOrm()
	user := models.User{Id: userId}
	err = o.Read(&user)
	if err != nil {
		fmt.Println("激活业务：根据用户id查询用户失败")
		this.Data["errmsg"] = "要激活的用户不存在"
		this.TplName = "register.html"
		return
	}
	user.Active = true
	_, err = o.Update(&user)
	if err != nil {
		fmt.Println("激活业务：更新用户激活转态出错")
		this.Data["errmsg"] = "要激活的用户不存在"
		this.TplName = "register.html"
		return
	}
	//	激活成功，重定向到登录界面
	this.Redirect("login.html", 302)
}

//显示登录页面
func (this *UserController) ShowLogin() {
	//获取用户记录用户名的转态
	username := this.Ctx.GetCookie("username")
	//解码
	temp, err := base64.StdEncoding.DecodeString(username)
	if err != nil {
		fmt.Println("显示登录页面报错：获取cookile解码用户名出错")
		this.TplName = "login.html"
		return
	}
	if string(temp) == "" {
		this.Data["username"] = ""
		this.Data["checked"] = ""
	} else {
		this.Data["username"] = string(temp)
		this.Data["checked"] = "checked"
	}
	this.TplName = "login.html"
}

//登录业务
func (this *UserController) HandleLogin() {
	//1. 获取数据
	username := this.GetString("username")
	password := this.GetString("pwd")
	allow := this.GetString("allow")
	//2. 校验数据
	if username == "" || password == "" {
		fmt.Println("登录业务：用户密码或密码为空")
		this.Data["errmsg"] = "登录失败，用户密码或密码为空"
		this.TplName = "login.html"
		return
	}
	//3. 根据用户名查询数据库
	o := orm.NewOrm()
	user := models.User{UserName: username}
	err := o.Read(&user, "UserName")
	if err != nil {
		//查询失败，用户没有被注册
		fmt.Println("登录业务：根据用户名查询用户失败")
		this.Data["errmsg"] = "登录失败，用户没有被注册"
		this.TplName = "login.html"
		return
	}
	//校验用户处于激活状态
	if user.Active != true {
		fmt.Println("登录业务：用户未激活")
		this.Data["errmsg"] = "登录失败，用户没有激活"
		this.TplName = "login.html"
		return
	}
	if password != user.PassWord {
		fmt.Println("登录业务：密码比对错误")
		this.Data["errmsg"] = "登录失败，用户或密码错误"
		this.TplName = "login.html"
		return
	}
	//4. 处理记住用户名功能
	if allow == "on" {
		//	记住用户名 -- 使用浏览器中的存储空间
		//base64加密
		temp := base64.StdEncoding.EncodeToString([]byte(username))
		this.Ctx.SetCookie("username", temp, 100)
	} else {
		this.Ctx.SetCookie("username", username, -1) // 时间设置为-1 立即失效相当于删除cookieuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuul
	}
	//5. 将登录成功的用户信息保存在session中
	//将登录成功的用户保存在session中 -- session是存储在服务端内存中，需要在beego配置文件中开启session 使session生效
	this.SetSession("username", user.UserName)
	this.Redirect("/index", 302)
}

//退出登录业务
func (this *UserController) HandleLoginOut() {
	//	删除session中保存的用户信息
	this.DelSession("username")
	this.Redirect("/index", 302)
}

//用户中心业务
//展示修改用户地址路由
func (this *UserController) ShowUserSite() {
	GetUser(&this.Controller)
	//获取用户默认地址
	o := orm.NewOrm()
	address := models.Address{}
	o.QueryTable("Address").RelatedSel("User").Filter("Isdefault", true).One(&address)
	// 将查询到的地址返回视图
	this.Data["address"] = address
	this.Layout = "userCenterLayout.html"
	this.TplName = "user_center_site.html"
}

//添加用户地址业务
func (this *UserController) HandleUserSite() {
	GetUser(&this.Controller)
	//	获取数据
	receiver := this.GetString("receiver")
	userAddress := this.GetString("userAddress")
	zipCode := this.GetString("zipCode")
	phone := this.GetString("phone")
	//	校验数据 --手机号和邮编可以使用正则进行校验这略
	if receiver == "" || userAddress == "" || zipCode == "" || phone == "" {
		fmt.Print("用户中心报错：添加用户地址的相关信息为空")
		this.Data["errmsg"] = "数据不完整，请重新添加地址"
		this.Layout = "userCenterLayout.html"
		this.TplName = "user_center_site.html"
		return
	}
	//	将数据插入数据库 -- 由于用户与用户地址是一对多的关系，因此在插入地址时也要提供用户信息
	o := orm.NewOrm()
	username := this.GetSession("username")
	var user models.User
	if username == nil {
		this.Data["errmsg"] = "您未登录，不能添加地址"
		this.Layout = "userCenterLayout.html"
		this.TplName = "user_center_site.html"
		return
	} else {
		user.UserName = username.(string)
		err := o.Read(&user, "userName")
		if err != nil {
			fmt.Print("用户中心报错：根据用户地址查询用户失败", err)
			this.Data["errmsg"] = "您未登录，不能添加地址"
			this.Layout = "userCenterLayout.html"
			this.TplName = "user_center_site.html"
			return
		}
	}

	address := models.Address{Isdefault: true}
	//插入前先判断是否有默认地址 -- 默认地址要在当前页显示且默认地址只能有一个，为了方便默认每一次新添加的地址为默认地址
	err := o.Read(&address, "Isdefault") // 根据默认地址 字段查询数据库
	if err == nil {
		//	说明数据库中有默认地址，此时将数据库的默认地址改为false,新插入的地址设为默认地址
		fmt.Println("数据库有默认地址：", address)
		address.Isdefault = false // 上面是取地址--所以里面保存有查询到的默认地址
		_, err = o.Update(&address)
		if err != nil {
			fmt.Print("用户中心报错：更新默认地址用户失败", err)
			this.Layout = "userCenterLayout.html"
			this.TplName = "user_center_site.html"
			return
		}
		//o.Update(&address)
		var address_add models.Address
		address_add.Addr = userAddress
		address_add.Isdefault = true
		address_add.Phone = phone
		address_add.Receiver = receiver
		address_add.User = &user
		//fmt.Println("数据库有默认地址修改后：", address_add)
		_, err = o.Insert(&address_add)
		if err != nil {
			fmt.Print("用户中心报错：添加用户地址失败", err)
			this.Data["errmsg"] = "添加用户地址失败，请重新添加地址"
			this.Layout = "userCenterLayout.html"
			this.TplName = "user_center_site.html"
			return
		}
	} else {
		//	数据库没有默认地址 -- 直接插入
		address_add := models.Address{
			Isdefault: true,
		}

		//address_add := models.Address{Isdefault: true}
		address_add.Receiver = receiver
		address_add.User = &user
		address_add.Phone = phone
		address_add.Zipcode = zipCode
		address.Addr = userAddress
		//fmt.Println("数据库没有默认地址", address_add.Isdefault, address_add.User, address.Addr)
		_, err = o.Insert(&address_add)
		if err != nil {
			fmt.Print("用户中心报错：添加用户地址失败", err)
			this.Data["errmsg"] = "添加用户地址失败，请重新添加地址"
			this.Layout = "userCenterLayout.html"
			this.TplName = "user_center_site.html"
			return
		}
	}
	//	重定向到 地址添加页面 -- 通过路由 因为在显示页面时还要查询数据库显示最小默认地址
	this.Redirect("/user/userSite", 302)

}

//展示用户订单页面
func (this *UserController) ShowUserOrder() {
	GetUser(&this.Controller)
	// 获取用户信息
	username := this.GetSession("username")
	if username == nil {
		fmt.Println("用户未登录")
		this.Redirect("/login", 302)
		return
	}
	o := orm.NewOrm()
	//	根据用户登录信息，查询用户详细
	user := models.User{UserName: username.(string)}
	err := o.Read(&user, "UserName")
	if err != nil {
		fmt.Println("展示订单业务报错，查询用户信息失败")
		this.Redirect("/login", 302)
		return
	}
	//	获取订单信息表  -- 当前用户的所有订单信息表
	var orderInfos []models.OrderInfo
	o.QueryTable("OrderInfo").RelatedSel("User").Filter("User__Id", user.Id).All(&orderInfos)
	//	存储返回页面数据容器
	var orderBuffer []map[string]interface{}
	orderBuffer = make([]map[string]interface{}, len(orderInfos))
	for index, orderInfo := range orderInfos {
		// 根据每一个点单信息，查询出关联的订单商品表,
		var orderGoods []models.OrderGoods
		o.QueryTable("OrderGoods").RelatedSel("OrderInfo", "GoodsSKU").Filter("OrderInfo__Id", orderInfo.Id).All(&orderGoods)
		// 定义一个与返回页面数据类型相同的临时变量，存放每一次循环的数据，并将其放入切片
		temp := make(map[string]interface{})
		temp["orderInfo"] = orderInfo
		temp["orderGoods"] = orderGoods
		orderBuffer[index] = temp
		fmt.Println(orderInfo)
	}
	//j将数据返回给视图
	this.Data["orderBuffer"] = orderBuffer
	this.Layout = "userCenterLayout.html"
	this.TplName = "user_center_order.html"
}

//展示用户个人信息页面
func (this *UserController) ShowUserInfo() {
	username := GetUser(&this.Controller)
	if username == "" {
		fmt.Println("用户为登录")
		this.Redirect("/index", 302) // -- 用户为登录，不允许执行下列代码
		return
	}
	//查询用户默认地址信息进行展示
	o := orm.NewOrm()
	address := models.Address{Isdefault: true}
	err := o.QueryTable("Address").RelatedSel("User").Filter("Isdefault", true).One(&address)
	if err != nil {
		fmt.Print("用户中心报错：未查询到默认地址", err)
		this.Data["errmsg"] = "您没有默认地址，快去添加一个吧"
		this.Layout = "userCenterLayout.html"
		this.TplName = "user_center_site.html"
		return
	}
	//查询用户历史浏览记录
	user := models.User{UserName: username}
	o.Read(&user, "UserName")
	//连接redis数据库
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		fmt.Println("个人信息中心报错：连接redis数据库报错", err)
	}
	defer conn.Close()                                                    // 及时关闭连接 -- 避免内存泄漏
	res, err := conn.Do("lrange", "history_"+strconv.Itoa(user.Id), 0, 4) // 只能用lrange -- 因为取多个，不能使用lpop：取一个
	if err != nil {
		fmt.Println("从redis中获取数据报错", err)
	}
	//回复助手函数
	goodsId, _ := redis.Ints(res, err)  // 多个数转为int类型  redis.Ints
	var history_goods []models.GoodsSKU //切面，里面的每一个数据类型均为 GoodsSKU
	//将查询到的商品id赋值给 history_goods
	for _, val := range goodsId {
		var goods models.GoodsSKU
		goods.ID = val
		o.Read(&goods, "ID")
		history_goods = append(history_goods, goods)
	}
	this.Data["history_goods"] = history_goods
	this.Data["username"] = username
	this.Data["address"] = address
	this.Layout = "userCenterLayout.html"
	this.TplName = "user_center_info.html"
}
