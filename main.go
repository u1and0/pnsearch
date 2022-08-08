package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/tobgu/qframe"
	qsql "github.com/tobgu/qframe/config/sql"
)

const (
	// VERSION : schd version
	VERSION  = "v0.1.0"
	FILENAME = "./test/test50row.db"
	//  "/mnt/data/sqlite3.db"
)

var (
	showVersion bool
	qf          qframe.QFrame
)

type (
	Table []Row
	// Table struct {
	// OrderNo string// 製番
	// ProductNo string// 製番_品名
	// UnitNo string // ユニットNo
	// Pid string // 品番
	// Name string // 品名
	// Type string // 形式寸法
	// Unit string // 単位
	// Quantity int // 仕入原価数量
	// // 仕入原価単価
	// // 入原価金額
	// // 在庫払出数量
	// // 在庫払出単価
	// // 在庫払出金額
	// // 登録日
	// // 発注日
	// // 納期
	// // 回答納期
	// // 納入日
	// // 発注区分
	// // メーカ
	// Product string // 材質
	// /Quantity int / 員数
	// // 必要数
	// // 部品発注数
	// // 発注残数
	// // 発注単価
	// // 発注金額
	// // 進捗レベル
	// // 工程名
	// // 仕入先略称
	// // オーダーNo
	// // 納入場所名
	// // 部品備考
	// // 原価費目ｺｰﾄﾞ
	// // 原価費目名
	// 		OrderNum string
	// 		ProductNum
	// Pid  sql.NullString
	// Name sql.NullString
	// Type sql.NullString
	// }
	Row struct {
		Pid  string
		Name string
		Type string
	}
	Query struct {
		Pid  string `form:"品番"`
		Name string `form:"品名"`
		Type string `form:"形式寸法"`
	}
)

// Show version
func init() {
	flag.BoolVar(&showVersion, "v", false, "Show version")
	flag.Parse()
	if showVersion {
		fmt.Println("schd version", VERSION)
		os.Exit(0) // Exit with version info
	}
}

// DB in memory
func init() {
	db, err := sql.Open("sqlite3", FILENAME)
	if err != nil {
		log.Fatal(err)
	}
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	qf = qframe.ReadSQL(tx, qsql.Query("select * from order2"), qsql.SQLite())
}

func main() {
	// Router
	r := gin.Default()
	r.Static("/static", "./static")
	r.LoadHTMLGlob("template/*.tmpl")

	// API
	r.GET("/", func(c *gin.Context) {
		view := []qframe.StringView{
			qf.MustStringView("品番"),
			qf.MustStringView("品名"),
			qf.MustStringView("形式寸法"),
		}
		table := Table{}
		pid := toSlice(view[0])
		name := toSlice(view[1])
		typed := toSlice(view[2])
		for i := 0; i < len(name); i++ {
			if i > 1000 { // 最大1000件表示
				break
			}
			r := Row{
				Pid:  pid[i],
				Name: name[i],
				Type: typed[i],
			}
			table = append(table, r)
		}
		fmt.Println(table)
		c.HTML(http.StatusOK, "table.tmpl", gin.H{
			"msg":   "トップから10件を表示",
			"table": table,
		})
	})

	// r.GET("/search", func(c *gin.Context) {
	// 	table := Table{}
	// 	q := new(Query)
	// 	if err := c.ShouldBind(q); err != nil {
	// 		c.HTML(http.StatusBadRequest, "table.tmpl", gin.H{
	// 			"msg": "Bad Query",
	// 		})
	// 		return
	// 	}
	// 	fmt.Printf("%+v\n", q)
	// 	if len(table.品番) == 0 {
	// 		c.HTML(http.StatusBadRequest, "table.tmpl", gin.H{
	// 			"msg": "検索結果がありません",
	// 		})
	// 		return
	// 	}
	// 	c.HTML(http.StatusOK, "table.tmpl", gin.H{
	// 		"msg":   fmt.Sprintf("%#v を検索, 30件を表示", q),
	// 		"table": q.search(),
	// 	})
	// })

	r.Run()
}

// func (q *Query) search() (result Table) {
// 	table := Table{}
// 	for _, r := range table {
// 		bol := strings.Contains(r.Pid, q.Pid) &&
// 			strings.Contains(r.Name, q.Name) &&
// 			strings.Contains(r.Type, q.Type)
// 		if bol {
// 			result = append(result, r)
// 			fmt.Println(r)
// 		}
// 	}
// 	return
// }
//
func toString(ptr *string) string {
	if (ptr == nil) || reflect.ValueOf(ptr).IsNil() {
		return ""
	}
	return *ptr
}

func toSlice(view qframe.StringView) (stringSlice []string) {
	for _, v := range view.Slice() {
		stringSlice = append(stringSlice, toString(v))
	}
	return
}
