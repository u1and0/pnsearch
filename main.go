package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/tobgu/qframe"
	qsql "github.com/tobgu/qframe/config/sql"
)

const (
	// VERSION : version info
	VERSION = "v0.1.0"
	// FILENAME = "./test/test50row.db"
	FILENAME = "/mnt/2_Common/06_ツール/software/python/jupyter/PNsearch/data/sqlite3.db"
	// SQLQ : 実行するSQL文
	SQLQ = `SELECT
			製番,
			ユニットNo,
			品番,
			品名,
			形式寸法,
			単位,
			メーカ,
			必要数,
			部品発注数
			FROM order2
			WHERE rowid > 800000
			`
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
		UnitNo string
		Pid    string
		Name   string
		Type   string
	}
	Query struct {
		UnitNo string `form:"要求番号"`
		Pid    string `form:"品番"`
		Name   string `form:"品名"`
		Type   string `form:"形式寸法"`
	}
)

// Show version
func init() {
	flag.BoolVar(&showVersion, "v", false, "Show version")
	flag.Parse()
	if showVersion {
		fmt.Println("pnsearch version", VERSION)
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
	qf = qframe.ReadSQL(tx, qsql.Query(SQLQ), qsql.SQLite())
	fmt.Println("qframe:", qf)
}

func main() {
	// Router
	r := gin.Default()
	r.Static("/static", "./static")
	r.LoadHTMLGlob("template/*.tmpl")

	// API
	r.GET("/", func(c *gin.Context) {
		table := Frame2Table(qf)
		fmt.Println(table)
		c.HTML(http.StatusOK, "table.tmpl", gin.H{
			"msg":   "トップから10件を表示",
			"table": table,
		})
	})

	r.GET("/search", func(c *gin.Context) {
		q := new(Query)
		if err := c.ShouldBind(q); err != nil {
			c.HTML(http.StatusBadRequest, "table.tmpl", gin.H{
				"msg": "Bad Query",
			})
			return
		}
		fmt.Printf("query: %#v\n", q)

		// Search word
		filtered := q.search()
		table := Frame2Table(filtered)
		if len(table) == 0 {
			c.HTML(http.StatusBadRequest, "table.tmpl", gin.H{
				"msg": "検索結果がありません",
			})
			return
		}
		c.HTML(http.StatusOK, "table.tmpl", gin.H{
			"msg":   fmt.Sprintf("%#v を検索, %d件を表示", q, len(table)),
			"table": table,
		})
	})

	r.Run()
}

// filterConstructor : 正規表現で検索するfilterを生成
// func filterConstructor(q, col string) qframe.Filter {
// 	re := regexp.MustCompile(ToRegex(q))
// 	return qframe.Filter{
// 		Comparator: func(p *string) bool { return re.MatchString(toString(p)) },
// 		Column:     col,
// 	}
// }

func (q *Query) search() qframe.QFrame {
	res := []*regexp.Regexp{
		regexp.MustCompile(ToRegex(q.UnitNo)),
		regexp.MustCompile(ToRegex(q.Pid)),
		regexp.MustCompile(ToRegex(q.Name)),
		regexp.MustCompile(ToRegex(q.Type)),
	}

	filters := []qframe.FilterClause{
		qframe.Filter{
			Comparator: func(p *string) bool { return res[0].MatchString(toString(p)) },
			Column:     "ユニットNo",
		},
		qframe.Filter{
			Comparator: func(p *string) bool { return res[1].MatchString(toString(p)) },
			Column:     "品番",
		},
		qframe.Filter{
			Comparator: func(p *string) bool { return res[2].MatchString(toString(p)) },
			Column:     "品名",
		},
		qframe.Filter{
			Comparator: func(p *string) bool { return res[3].MatchString(toString(p)) },
			Column:     "形式寸法",
		},
		// filterConstructor(q.Pid, "品番"),
		// filterConstructor(q.Name, "品名"),
		// filterConstructor(q.Type, "形式寸法"),
	}

	// re := regexp.MustCompile(ToRegex(q.Name)) //`.*A.*00.*`)
	// filter := qframe.Filter{
	// 	Comparator: func(p *string) bool { return re.MatchString(toString(p)) },
	// 	Column:     "品名",
	// }
	// re2 := regexp.MustCompile(ToRegex(q.Pid)) //`.*GAA.*`)
	// filter2 := qframe.Filter{
	// 	Comparator: func(p *string) bool { return re2.MatchString(toString(p)) },
	// 	Column:     "品番",
	// }
	fmt.Println(filters)
	return qf.Filter(qframe.And(filters...))
}

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

// ToRegex : スペース区切りを正規表現.*で埋める
func ToRegex(s string) string {
	r := strings.Join(strings.Split(s, " "), `.*`)
	return fmt.Sprintf(`.*%s.*`, r)
}

// Frame2Table : QFrame をTableへ変換
func Frame2Table(qf qframe.QFrame) (table Table) {
	view := []qframe.StringView{
		qf.MustStringView("ユニットNo"),
		qf.MustStringView("品番"),
		qf.MustStringView("品名"),
		qf.MustStringView("形式寸法"),
	}
	slices := [][]string{
		toSlice(view[0]),
		toSlice(view[1]),
		toSlice(view[2]),
		toSlice(view[3]),
	}
	// NameとTypeは常に表示する仕様
	for i := 0; i < len(slices[2]); i++ {
		if i >= 1000 { // 最大1000件表示
			break
		}
		r := Row{
			UnitNo: slices[0][i],
			Pid:    slices[1][i],
			Name:   slices[2][i],
			Type:   slices[3][i],
		}
		table = append(table, r)
	}
	return
}
