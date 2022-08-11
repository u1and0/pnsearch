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
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/tobgu/qframe"
	qsql "github.com/tobgu/qframe/config/sql"
)

const (
	// VERSION : version info
	VERSION = "v0.1.2r"
	// FILENAME = "./test/test50row.db"
	FILENAME = "./data/sqlite3.db"
	// SQLQ : 実行するSQL文
	SQLQ = `SELECT
			*
			FROM order2
			ORDER BY 発注日
			`
	// LIMIT 1000
	// WHERE rowid > 800000

	// MAXROW : qfからTableへ変換する最大行数
	MAXROW = 1000
)

var (
	showVersion bool
	debug       bool
	allData     qframe.QFrame
	portnum     int
	filename    string
)

type (
	// Table : HTMLへ書き込むための行指向のstruct
	Table []Row
	// Column : toSlice()で変換されるqfの列
	Column []string
	// Row : Tableの一行
	Row struct {
		ReceivedOrderNo   int16  // 受注No
		ProductNo         string // 製番
		ProductNoName     string // 製番_品名
		UnitNo            string // ユニットNo
		Pid               string // 品番
		Name              string // 品名
		Type              string // 形式寸法
		Unit              string // 単位
		PurchaseQuantity  string // 仕入原価数量
		PurchaseUnitPrice string // 仕入原価単価
		PurchaseCost      string // 仕入原価金額
		StockQuantity     string // 在庫払出数量
		StockUnitPrice    string // 在庫払出単価
		StockCost         string // 在庫払出金額
		RecordDate        string // 登録日
		OrderDate         string // 発注日
		DeliveryDate      string // 納期
		ReplyDeliveryDate string // 回答納期
		RealDeliveryDate  string // 納入日
		OrderDivision     string // 発注区分
		Maker             string // メーカ
		Material          string // 材質
		Quantity          string // 員数
		OrderQuantity     string // 必要数
		OrderNum          string // 部品部品発注数
		OrderRest         string // 発注残数
		OrderUnitPrice    string // 発注単価
		OrderCost         string // 発注金額
		ProgressLevel     string // 進捗レベル
		Process           string // 工程名
		Vendor            string // 仕入先略称
		OrderNo           string // オーダーNo
		DeliveryPlace     string // 納入場所名
		Misc              string // 部品備考
		CostCode          string // 原価費目ｺｰﾄﾞ
		CostName          string // 原価費目名
	}

	/* テーブル情報
	検索、ソートのことは考えず
	表示とコーディングしやすさのことを考慮して、
	すべてTEXT型に変更した。
		CREATE TABLE order2 (
		"index" INTEGER,
		  "受注No" TEXT,
		  "製番" TEXT,
		  "製番_品名" TEXT,
		  "ユニットNo" TEXT,
		  "品番" TEXT,
		  "品名" TEXT,
		  "形式寸法" TEXT,
		  "単位" TEXT,
		  "仕入原価数量" TEXT,
		  "仕入原価単価" TEXT,
		  "仕入原価金額" TEXT,
		  "在庫払出数量" TEXT,
		  "在庫払出単価" TEXT,
		  "在庫払出金額" TEXT,
		  "登録日" TEXT,
		  "発注日" TEXT,
		  "納期" TEXT,
		  "回答納期" TEXT,
		  "納入日" TEXT,
		  "発注区分" TEXT,
		  "メーカ" TEXT,
		  "材質" TEXT,
		  "員数" TEXT,
		  "必要数" TEXT,
		  "部品発注数" TEXT,
		  "発注残数" TEXT,
		  "発注単価" TEXT,
		  "発注金額" TEXT,
		  "進捗レベル" TEXT,
		  "工程名" TEXT,
		  "仕入先略称" TEXT,
		  "オーダーNo" TEXT,
		  "納入場所名" TEXT,
		  "部品備考" TEXT,
		  "原価費目ｺｰﾄﾞ" TEXT,
		  "原価費目名" TEXT
		);
	*/

	// Query : URLクエリパラメータ
	Query struct {
		ProductNo string `form:"製番"`
		UnitNo    string `form:"要求番号"`
		Pid       string `form:"品番"`
		Name      string `form:"品名"`
		Type      string `form:"形式寸法"`
		Maker     string `form:"メーカ"`
		Vendor    string `form:"仕入先"`
	}
)

// Show version
func init() {
	flag.BoolVar(&showVersion, "v", false, "Show version")
	flag.BoolVar(&debug, "debug", false, "Run debug mode")
	flag.IntVar(&portnum, "p", 9000, "Access port")
	flag.StringVar(&filename, "f", FILENAME, "SQL database file path")
	flag.Parse()
	if showVersion {
		fmt.Println("pnsearch version", VERSION)
		os.Exit(0) // Exit with version info
	}
	if !debug {
		gin.SetMode(gin.ReleaseMode)
	}
}

// DB in memory
// const のSQLQで読み込まれる全データをqf に読み込む。
func init() {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		log.Fatal(err)
	}
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	allData = qframe.ReadSQL(tx, qsql.Query(SQLQ), qsql.SQLite())
	log.Println("qframe:", allData)
}

func main() {
	// Router
	r := gin.Default()
	r.Static("/static", "./static")
	r.LoadHTMLGlob("template/*.tmpl")

	// API
	r.GET("/", func(c *gin.Context) {
		table := T(allData)
		if debug {
			log.Println(table)
		}
		c.HTML(http.StatusOK, "table.tmpl", gin.H{
			"msg":   fmt.Sprintf("テストページ / トップから%d件を表示", len(table)),
			"table": table,
		})
	})

	r.GET("/search", func(c *gin.Context) {
		q := new(Query)
		if err := c.ShouldBind(q); err != nil {
			c.HTML(http.StatusBadRequest, "table.tmpl", gin.H{
				"msg": fmt.Sprintf("%#v Bad Query", q),
			})
			return
		}
		log.Println(fmt.Sprintf("query: %#v", q))

		// Search keyword by query parameter
		filtered := q.search()
		table := T(filtered)
		if len(table) == 0 {
			c.HTML(http.StatusBadRequest, "table.tmpl", gin.H{
				"msg": fmt.Sprintf("%#v を検索, 検索結果がありません", q),
			})
			return
		}
		c.HTML(http.StatusOK, "table.tmpl", gin.H{
			"msg":   fmt.Sprintf("%#v を検索, %d件中%d件を表示", q, filtered.Len(), len(table)),
			"table": table,
		})
	})

	port := ":" + strconv.Itoa(portnum)
	r.Run(port)
}

func (q *Query) search() qframe.QFrame {
	res := map[string]*regexp.Regexp{
		"製番":   regexp.MustCompile(ToRegex(q.ProductNo)),
		"要求番号": regexp.MustCompile(ToRegex(q.UnitNo)),
		"品番":   regexp.MustCompile(ToRegex(q.Pid)),
		"品名":   regexp.MustCompile(ToRegex(q.Name)),
		"形式寸法": regexp.MustCompile(ToRegex(q.Type)),
		"メーカ":  regexp.MustCompile(ToRegex(q.Maker)),
		"仕入先":  regexp.MustCompile(ToRegex(q.Vendor)),
	}

	filters := []qframe.FilterClause{
		qframe.Filter{
			Comparator: func(p *string) bool { return res["製番"].MatchString(toString(p)) },
			Column:     "製番",
		},
		qframe.Filter{
			Comparator: func(p *string) bool { return res["要求番号"].MatchString(toString(p)) },
			Column:     "ユニットNo",
		},
		qframe.Filter{
			Comparator: func(p *string) bool { return res["品番"].MatchString(toString(p)) },
			Column:     "品番",
		},
		qframe.Filter{
			Comparator: func(p *string) bool { return res["品名"].MatchString(toString(p)) },
			Column:     "品名",
		},
		qframe.Filter{
			Comparator: func(p *string) bool { return res["形式寸法"].MatchString(toString(p)) },
			Column:     "形式寸法",
		},
		qframe.Filter{
			Comparator: func(p *string) bool { return res["メーカ"].MatchString(toString(p)) },
			Column:     "メーカ",
		},
		qframe.Filter{
			Comparator: func(p *string) bool { return res["仕入先"].MatchString(toString(p)) },
			Column:     "仕入先略称",
		},
	}
	return allData.Filter(qframe.And(filters...))
}

func toString(ptr *string) string {
	if (ptr == nil) || reflect.ValueOf(ptr).IsNil() {
		return ""
	}
	return *ptr
}

func toSlice(qf qframe.QFrame, colName string) (stringSlice []string) {
	view := qf.MustStringView(colName)
	for _, v := range view.Slice() {
		stringSlice = append(stringSlice, toString(v))
	}
	return
}

// ToRegex : スペース区切りを正規表現.*で埋める
// (?i) for ignore case
// .* for any string
func ToRegex(s string) string {
	s = strings.ReplaceAll(s, "　", " ")       // 全角半角変換
	s = strings.ReplaceAll(s, "\t", " ")      // タブ文字削除
	s = strings.TrimSpace(s)                  // 左右の空白削除
	s = strings.Join(strings.Fields(s), `.*`) // スペースを.*に変換
	return fmt.Sprintf(`(?i).*%s.*`, s)
}

// T : QFrame をTableへ変換
func T(qf qframe.QFrame) (table Table) {
	slices := map[string]Column{}
	for _, k := range []string{"ユニットNo", "品番", "品名", "形式寸法",
		"メーカ", "材質", "工程名", "納入場所名", "発注単価", "発注金額",
		"発注日", "納入日"} {
		slices[k] = toSlice(qf, k)
	}

	// NameとTypeは常に表示する仕様
	for i := 0; i < len(slices["品名"]); i++ {
		if i >= MAXROW { // 最大1000件表示
			break
		}
		r := Row{
			UnitNo:           slices["ユニットNo"][i],
			Pid:              slices["品番"][i],
			Name:             slices["品名"][i],
			Type:             slices["形式寸法"][i],
			Maker:            slices["メーカ"][i],
			Material:         slices["材質"][i],
			Process:          slices["工程名"][i],
			DeliveryPlace:    slices["納入場所名"][i],
			OrderUnitPrice:   slices["発注単価"][i],
			OrderCost:        slices["発注金額"][i],
			OrderDate:        slices["発注日"][i],
			RealDeliveryDate: slices["納入日"][i],
		}
		table = append(table, r)
	}
	return
}
