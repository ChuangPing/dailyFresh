package models

import (
	"fmt"
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql"
	"time"
)

//用户表
type User struct {
	Id        int          //默认为主键
	UserName  string       `orm:"size(20);unique"` //用户名  --不能重复
	PassWord  string       `orm:"size(20)"`        //用户密码
	Email     string       `orm:"size(50)"`        //用户邮箱
	Active    bool         `orm:"default(false)"`  //用户是否激活 -- 默认为false（邮件激活功能）
	Power     int          `orm:"default(0)"`      //用户权限设置 --默认0 未激活  1：激活状态
	Address   []*Address   `orm:"reverse(many)"`   //用户地址与地址表是一对多关系 -- 一个用户有多个地址
	Orderinfo []*OrderInfo `orm:"reverse(many)"`   //用户订单与订单表为一对多关系
}

//地址表
type Address struct {
	Id        int
	Receiver  string       `orm:"size(20)"`       //收件人
	Addr      string       `orm:"size(50)"`       //收件地址
	Zipcode   string       `orm:"size(20)"`       //邮编
	Phone     string       `orm:"size(20)"`       //联系方式
	Isdefault bool         `orm:"default(false)"` //是否为默认收货地址 false 0 非默认  true 1 默认
	User      *User        `orm:"rel(fk)"`        //与用户主键关联 多对一
	OrderInfo []*OrderInfo `orm:"reverse(many)"`  //订单信息表 一对多关系  -- 一个地址有多个订单
}

//商品表SPU 表  -- SPU 商品的大概信息
type Goods struct {
	Id       int
	Name     string      `orm:"size(20)"`      //商品名称
	Detail   string      `orm:"size(200)"`     //商品详情/描述
	GoodsSKU []*GoodsSKU `orm:"reverse(many)"` // SPU与SKU是一对多关系 一个SPU有多个SKU
}

//商品类型表
type GoodsType struct {
	Id                   int
	Name                 string                  //商品种类名称
	Logo                 string                  //logo
	Image                string                  //商品图片
	GoodsSKU             []*GoodsSKU             `orm:"reverse(many)"`
	IndexTypeGoodsBanner []*IndexTypeGoodsBanner `orm:"reverse(many)"`
}

//商品SKU表  -- SKU 每件商品的详细信息
type GoodsSKU struct {
	ID                   int
	Goods                *Goods                  `orm:"rel(fk)"`  // 商品的SPU表 与主键关联
	GoodsType            *GoodsType              `orm:"rel(fk)"`  // 商品的类型表
	Name                 string                  `orm:"size(20)"` // 商品名称
	Desc                 string                  //商品简介
	Price                int                     //商品价格
	Unite                string                  //商品单位
	Image                string                  //商品图片
	Stock                int                     `orm:"default(0)"`    //商品库存
	Sale                 int                     `orm:"default(1)"`    //商品销售数量
	Status               int                     `orm:"default(1)"`    //商品状态
	Time                 time.Time               `orm:"auto_now_add"`  //添加时间
	GoodsImage           []*GoodsImage           `orm:"reverse(many)"` //商品图片 -- 一个商品有多张图片
	IndexGoodsBanner     []*IndexGoodsBanner     `orm:"reverse(many)"`
	IndexTypeGoodsBanner []*IndexTypeGoodsBanner `orm:"reverse(many)"` //
	OrderGoods           []*OrderGoods           `orm:"reverse(many)"`
}

//商品图片表
type GoodsImage struct {
	Id       int
	Image    string    //商品图片
	GoodsSku *GoodsSKU `orm:"rel(fk)"` // 商品SKU 对应商品详细表的主键 他们的关系为 一对多
}

//首页轮播商品展示表
type IndexGoodsBanner struct {
	Id       int
	GoodsSku *GoodsSKU `orm:"rel(fk)"`
	Image    string    //商品图片 -- 轮播展示图片
	Index    int       `orm:"default(0)"` //展示顺序 默认为0
}

//首页商品分类展示表
type IndexTypeGoodsBanner struct {
	Id          int
	GoodsType   *GoodsType `orm:"rel(fk)"`    // 商品类型表
	GoodsSKU    *GoodsSKU  `orm:"rel(fk)"`    // 商品SKU表
	DisplayType int        `orm:"default(1)"` //展示类型 0：代表文字 1：代表图片
	Index       int        `orm:"default(0)" // 展示顺序`
}

//首页促销商品展示
type IndexPromotionBanner struct {
	Id    int
	Name  string `orm:"size(20)"` //活动名称
	Url   string `orm:"size(50)"` // 活动链接
	Image string // 图片
	Index int    `orm:"default(0)"` //展示顺序 默认为 0
}

//订单表
type OrderInfo struct {
	Id           int
	OrderId      string        `orm:"unique"`     // 订单iD 不允许重复
	User         *User         `orm:"rel(fk)"`    // 一个用户有多个订单 -- 用户的主键（1）与订单表的一个外键（非主键） 关联
	Address      *Address      `orm:"rel(fk)"`    // 用户地址
	PayMethod    int           `orm:"default(0)"` // 付款方式
	TotalCount   int           `orm:"default(1)"` //商品总数量 默认为1
	TotalPrice   int           //商品总价 -- 在下订单就计算好，存入数据库
	TransitPrice int           //运费
	OrderStatus  int           `orm:"default(1)"`    //订单状态
	TradeNo      string        `orm:"default('')"`   //支付编号
	Time         time.Time     `orm:"auto_now_add"`  //评论时间
	orderGoods   []*OrderGoods `orm:"reverse(many)"` //订单物品
}

//订单商品表
type OrderGoods struct {
	Id        int
	OrderInfo *OrderInfo `orm:"rel(fk)"`    //订单信息表
	GoodsSKU  *GoodsSKU  `orm:"rel(fk)"`    //商品表
	Count     int        `orm:"default(1)"` //商品数量
	Price     int        //商品价格
	Comment   string     `orm:"default('')"` //商品评论
}

//创建表初始化函数
func init() {
	//1. set default database 使用到了mysql驱动，因此需要下载go的mysql，使用下划线的方式导入，执行一下驱动中的init函数即可
	err := orm.RegisterDataBase("default", "mysql", "root:root@tcp(127.0.0.1:3306)/dailyfresh?charset=utf8")
	if err != nil {
		fmt.Println("连接数据库失败")
		return
	}
	//2. Register Model
	orm.RegisterModel(new(User), new(Address), new(OrderGoods), new(OrderInfo), new(IndexPromotionBanner), new(IndexTypeGoodsBanner), new(IndexGoodsBanner), new(GoodsImage), new(GoodsSKU), new(GoodsType), new(Goods))
	//3.create table  -- 参数固定  第二个参数：false 表存在不会创建
	orm.RunSyncdb("default", false, true)
}
