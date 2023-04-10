package routers

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
	"project_dailyfresh/controllers"
)

func init() {
	//  路由拦截
	beego.InsertFilter("/user/*", beego.BeforeExec, filterFunction)
	beego.Router("/", &controllers.MainController{})
	//	注册路由
	beego.Router("/register", &controllers.UserController{}, "get:ShowRegister;post:HandleRegister")
	//	激活注册账号路由
	beego.Router("/active", &controllers.UserController{}, "get:HandleActive")
	//	登录路由
	beego.Router("/login", &controllers.UserController{}, "get:ShowLogin;post:HandleLogin")
	//  退出登录路由
	beego.Router("/user/loginOut", &controllers.UserController{}, "get:HandleLoginOut")
	// 首页路由
	beego.Router("/index", &controllers.GoodsController{}, "get:ShowIndex")
	// 用户个人信息路由
	beego.Router("/user/userCenterInfo", &controllers.UserController{}, "get:ShowUserInfo")
	// 用户添加地址路由
	beego.Router("/user/userSite", &controllers.UserController{}, "get:ShowUserSite;post:HandleUserSite")
	// 用户订单路由
	beego.Router("/user/userOrder", &controllers.UserController{}, "get:ShowUserOrder")
	//	商品详情路由
	beego.Router("/goodsDetail", &controllers.GoodsController{}, "get:HandleGoodsDetail")
	// 商品列表路由
	beego.Router("/goodsList", &controllers.GoodsController{}, "get:ShowGoodsList")
	// 商品搜索路由
	beego.Router("/search", &controllers.GoodsController{}, "post:HandleSearch")
	// 添加购物车路由
	beego.Router("/user/addCart", &controllers.GoodsCartController{}, "post:HandleAddCart")
	// 显示购物车页面
	beego.Router("/user/showCart", &controllers.GoodsCartController{}, "get:ShowCart")
	//	更新购物车数量
	beego.Router("/user/updateCartCount", &controllers.GoodsCartController{}, "post:HandleUpdateCartCount")
	//	删除购物车商品路由
	beego.Router("/user/deleteCartGoods", &controllers.GoodsCartController{}, "post:HandleDelCartGoods")
	//	显示订单页面路由
	beego.Router("/user/showOrder", &controllers.GoodsOrderController{}, "post:ShowOrder")
	//	添加订单路由
	beego.Router("/user/addOrder", &controllers.GoodsOrderController{}, "post:HandleAddOrder")
	//	支付路由
	beego.Router("/user/orderPay", &controllers.GoodsOrderController{}, "get:HandlePay")
	// 支付成功路由
	beego.Router("/user/payok", &controllers.GoodsOrderController{}, "get:HandlePayOk")
}

var filterFunction = func(ctx *context.Context) {
	username := ctx.Input.Session("username")
	if username == nil {
		ctx.Redirect(302, "/login") // 用户未登录不能访问 /user/* 的路由
		return
	}
}
