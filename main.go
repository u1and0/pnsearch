package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

const (
	// VERSION : schd version
	VERSION = "v0.1.0"
)

var (
	showVersion bool
)

type (
	Table       []Row
	NullableRow struct {
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

		Pid  sql.NullString
		Name sql.NullString
		Type sql.NullString
	}
	Row struct {
		Pid  string
		Name string
		Type string
	}
)

func search(keywd string) (table Table) {
	var (
		db, err = sql.Open("sqlite3", "../../data/sqlite3.db")
		sqlQ    = fmt.Sprintf(`SELECT %s, %s, %s
						FROM "order2"
						`, "品番", "品名", "形式寸法")
	)
	if keywd != "" {
		sqlQ += fmt.Sprintf(`WHERE ("品番" LIKE "%%%s%%")
				LIMIT 30
				`, keywd)
	} else {
		sqlQ += `LIMIT 30`
	}
	rows, err := db.Query(sqlQ)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	// Null処理
	// nullVal := new(sql.NullString)
	for rows.Next() {
		// var pid, name string
		r := new(NullableRow)
		if err := rows.Scan(&r.Pid, &r.Name, &r.Type); err != nil {
			log.Fatal(err)
		}

		rv := Row{
			Pid:  r.Pid.String,
			Name: r.Name.String,
			Type: r.Type.String,
		}

		fmt.Println(rv)
		table = append(table, rv)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	return
}

func main() {
	flag.BoolVar(&showVersion, "v", false, "Show version")
	flag.Parse()
	if showVersion {
		fmt.Println("schd version", VERSION)
		os.Exit(0) // Exit with version info
	}

	// Router
	r := gin.Default()
	r.Static("/static", "./static")
	r.LoadHTMLGlob("template/*.tmpl")

	// API
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "table.tmpl", gin.H{
			"msg":   "トップから30件を表示",
			"table": search(""),
		})
	})
	r.GET("/:pid", func(c *gin.Context) {
		pid := c.Param("pid")
		table := search(pid)
		if len(table) == 0 {
			c.HTML(http.StatusBadRequest, "table.tmpl", gin.H{
				"msg": "検索結果がありません",
			})
			return
		}
		c.HTML(http.StatusOK, "table.tmpl", gin.H{
			"msg":   fmt.Sprintf("品番 %s を検索, 30件を表示", pid),
			"table": table,
		})
	})

	r.Run()
}
