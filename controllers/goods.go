package controllers

import (
	"fmt"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"github.com/gomodule/redigo/redis"
	"math"
	"project_dailyfresh/models"
	"strconv"
)

type GoodsController struct {
	beego.Controller
}

//获取session中保存的用户名  -- 封装函数
func GetUser(this *beego.Controller) string { // 参数使用父类，所有在controller包下的文件均可以使用
	username := this.GetSession("username")
	if username == nil {
		this.Data["username"] = ""
		return " "
	} else {
		this.Data["username"] = username.(string)
		return username.(string)
	}
}

//封装函数  -- 显示公共部分，传递用户信息，传递商品类型
func ShowLayout(this *beego.Controller) {
	GetUser(this) // 获取用户信息
	//获取商品类型
	o := orm.NewOrm()
	var goodsType []models.GoodsType
	o.QueryTable("GoodsType").All(&goodsType)
	cartCount := GetCartCount(this) // 获取购物车数据
	this.Data["cartCount"] = cartCount
	this.Data["GoodsType"] = goodsType
	// 将数据传递给视图
	this.Layout = "indexLayout.html"
}

//封装分页函数 --
func PageTool(pageCount int, pageIndex int) []int {
	var pages []int
	if pageCount <= 5 {
		//pages = [1,2,..,pageCount]
		pages = make([]int, pageCount)
		for i, _ := range pages {
			pages[i] = i + 1
		}
	} else if pageIndex <= 3 {
		//pages := make([]int,5)
		pages = []int{1, 2, 3, 4, 5}
	} else if pageIndex > pageCount-3 {
		pages = []int{pageCount - 4, pageCount - 3, pageCount - 2, pageCount - 1, pageCount}
	} else {
		pages = []int{pageCount - 2, pageCount - 1, pageCount, pageCount + 1, pageCount + 2}
	}
	return pages
}

//显示首页路由
func (this *GoodsController) ShowIndex() {
	GetUser(&this.Controller)
	//获取数据进行展示
	o := orm.NewOrm()
	//获取类型数据
	var goodsType []models.GoodsType
	o.QueryTable("GoodsType").All(&goodsType)
	this.Data["GoodsType"] = goodsType
	//获取轮播图数据
	var indexGoodsBanner []models.IndexGoodsBanner
	o.QueryTable("IndexGoodsBanner").OrderBy("Index").All(&indexGoodsBanner)
	this.Data["indexGoodsBanner"] = indexGoodsBanner
	//获取促销商品
	var indexPromotionBanner []models.IndexPromotionBanner
	o.QueryTable("IndexPromotionBanner").OrderBy("Index").All(&indexPromotionBanner)
	this.Data["indexPromotionBanner"] = indexPromotionBanner
	//获取首页分类展示商品
	goods := make([]map[string]interface{}, len(goodsType)) // map 的key为string value 为interface
	//将类型赋值给map的key
	for index, value := range goodsType {
		temp := make(map[string]interface{})
		temp["type"] = value
		goods[index] = temp
	}
	for _, value := range goods {
		var textGoods []models.IndexTypeGoodsBanner
		var imgGoods []models.IndexTypeGoodsBanner
		//获取文字商品数据
		o.QueryTable("IndexTypeGoodsBanner").RelatedSel("GoodsType", "GoodsSKU").OrderBy("Index").Filter("GoodsType", value["type"]).Filter("DisplayType", 0).All(&textGoods)
		//获取图片商品数据
		o.QueryTable("IndexTypeGoodsBanner").RelatedSel("GoodsType", "GoodsSKU").OrderBy("Index").Filter("GoodsType", value["type"]).Filter("DisplayType", 1).All(&imgGoods)
		value["textGoods"] = textGoods
		value["imgGoods"] = imgGoods
	}
	this.Data["goods"] = goods
	cartCount := GetCartCount(&this.Controller) // 获取购物车数据
	this.Data["cartCount"] = cartCount
	this.Layout = "indexLayout.html"
	this.TplName = "index.html"
}

//展示商品详情业务
func (this *GoodsController) HandleGoodsDetail() {
	goodId, err := this.GetInt("id")
	if err != nil {
		fmt.Println("获取商品id时出错")
		this.Redirect("/index", 302)
		return
	}
	// 处理历史浏览记录  -- 使用redis数据库进行存储，由于历史浏览记录要求有顺序，因此使用set集合
	//1. 先判断用户是否登录
	username := GetUser(&this.Controller)
	if username != "" {
		o := orm.NewOrm()
		user := models.User{UserName: username}
		err = o.Read(&user, "userName")
		if err != nil {
			fmt.Println("查看商品详情报错：根据用户id查询用户信息报错") // -- 没有登录的用户也可查看商品详情
		}
		//连接：redis 中 key：用户id  value：商品id
		conn, err := redis.Dial("tcp", ":6379")
		if err != nil {
			fmt.Println("查看商品详情报错：连接redis数据库报错", err)
			//this.Redirect("/index", 302) -- 继续展示详情页 --
			//return //-- 错误
		} else {
			defer conn.Close() // 及时关闭
			//存入收据前，先删除用户浏览已添加的浏览记录  -- 因为用户可能多次浏览一个商品，而我们只会保存一次
			conn.Do("lrem", "history_"+strconv.Itoa(user.Id), 0, goodId)
			conn.Do("lpush", "history_"+strconv.Itoa(user.Id), goodId)
		}
	}

	//根据商品id查询商品详情表
	o := orm.NewOrm()
	goodSKU := models.GoodsSKU{ID: goodId}
	o.QueryTable("GoodsSKU").RelatedSel("GoodsType", "Goods").Filter("ID", goodId).One(&goodSKU)
	//获取同类型商品，且时间靠前的两个商品
	var newGoodSKU []models.GoodsSKU
	o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType", goodSKU.GoodsType).OrderBy("Time").Limit(2, 0).All(&newGoodSKU)
	this.Data["goodSKU"] = goodSKU
	this.Data["newGoodSKU"] = newGoodSKU
	ShowLayout(&this.Controller)
	this.TplName = "detail.html"
}

//展示商品列表页
func (this *GoodsController) ShowGoodsList() {
	typeId, err := this.GetInt("typeId")
	if err != nil {
		fmt.Println("列表页展示出错：", err)

	}
	ShowLayout(&this.Controller) // 展示模板
	//获取同类型商品的新品推荐 -- 2个
	o := orm.NewOrm()
	var goodsNew []models.GoodsSKU //新品商品 -- 只展示两条新品推荐商品
	o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id", typeId).Limit(2, 0).All(&goodsNew)
	//将推荐的新品传递到视图
	//fmt.Println("dddd:", goodsNew)
	this.Data["goodsNew"] = goodsNew
	//获取同类型的所有商品
	var goods []models.GoodsSKU
	//o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id", typeId).All(&goods) -- 查询所有商品
	//获取要显示数据的总数目 -- 查询数据库  -- GoodsSKU与GoodsType表通过GoodsType中的 Id 关联，因此数据库中会创建关联表，其中有GoodsType_Id 由于orm框架大写Id会加一个下划线变为 GoodsType__Id,过滤除了typeId=typeId的GoodsSKU商品
	count, _ := o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id", typeId).Count()

	//每页显示的数据数目 -- 每页显示三条数据
	pageSize := 3

	//总共的页数 -- 总数据/每页显示的数目
	pageCount := math.Ceil(float64(count) / float64(pageSize))

	//获取前端传第的当前页码 -- 需要做处理，因为当用户首次访问时默认为访问第一页
	pageIndex, err := this.GetInt("pageIndex")
	if err != nil {
		//用户首次访问list页面，默认访问第一页
		pageIndex = 1
	}
	pages := PageTool(int(pageCount), pageIndex)
	this.Data["pages"] = pages
	this.Data["typeId"] = typeId
	this.Data["pageIndex"] = pageIndex
	start := (pageIndex - 1) * pageSize
	//获取上一页页码
	prePage := pageIndex - 1
	if prePage == 1 {
		prePage = 1
	}
	this.Data["prePage"] = prePage
	//获取下一页页码
	nextPage := pageIndex + 1
	if nextPage > int(pageCount) {
		nextPage = int(pageCount)
	}
	this.Data["nextpage"] = nextPage
	//根据用户选择的排序方式，查询数据库
	sort := this.GetString("sort")
	if sort == "" {
		//	用户第一次进入list未选择排序方式
		o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id", typeId).Limit(pageSize, start).All(&goods)
		this.Data["sort"] = ""
		this.Data["goods"] = goods
	} else if sort == "price" {
		o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id", typeId).OrderBy("Price").Limit(pageSize, start).All(&goods)
		this.Data["sort"] = "price"
		this.Data["goods"] = goods
	} else if sort == "sale" {
		o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("goodsType__Id", typeId).OrderBy("Sale").Limit(pageSize, start).All(&goods)
		this.Data["sort"] = "sale"
		this.Data["goods"] = goods
	}
	//将获取到的对应类型全部商品传递给视图
	this.Data["goods"] = goods
	this.TplName = "list.html"
}

//搜索商品业务
func (this *GoodsController) HandleSearch() {
	//	获取数据
	search := this.GetString("search")
	o := orm.NewOrm()
	goods := []models.GoodsSKU{}
	//	校验数据
	if search == "" {
		//默认展示全部数据
		o.QueryTable("GoodsSKU").All(&goods)
		this.Data["goods"] = goods
		ShowLayout(&this.Controller)
		this.TplName = "search.html"
		return
	}
	o.QueryTable("GoodsSKU").Filter("Name__icontains", search).All(&goods)
	//	返回视图
	this.Data["goods"] = goods
	ShowLayout(&this.Controller)
	this.TplName = "search.html"
}
