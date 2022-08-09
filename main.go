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
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/tobgu/qframe"
	qsql "github.com/tobgu/qframe/config/sql"
)

const (
	// VERSION : version info
	VERSION = "v0.1.0"
	// FILENAME = "./test/test50row.db"
	FILENAME = "./data/sqlite3.db"
	// SQLQ : 実行するSQL文
	SQLQ = `SELECT
			製番,
			ユニットNo,
			品番,
			品名,
			形式寸法,
			単位,
			材質,
			メーカ,
			仕入先略称,
			工程名,
			必要数,
			部品発注数,
			発注単価,
			発注金額,
			納入場所名
			FROM order2
			ORDER BY 製番
			`
	// WHERE rowid > 800000

	// MAXROW : qfからTableへ変換する最大行数
	MAXROW = 1000
)

var (
	showVersion bool
	debug       bool
	qf          qframe.QFrame
	portnum     int
)

type (
	// Table : HTMLへ書き込むための行指向のstruct
	Table []Row
	// Column : toSlice()で変換されるqfの列
	Column []string
	// Columnf : toSlice()で変換されるqfの列
	Columnf []float64
	// Row : Tableの一行
	Row struct {
		ReceivedOrderNo   int16     // 受注No
		ProductNo         string    // 製番
		ProductNo_Name    string    // 製番_品名
		UnitNo            string    // ユニットNo
		Pid               string    // 品番
		Name              string    // 品名
		Type              string    // 形式寸法
		Unit              string    // 単位
		PurchaseQuantity  float64   // 仕入原価数量
		PurchaseUnitPrice float64   // 仕入原価単価
		PurchaseCost      float64   // 仕入原価金額
		StockQuantity     float64   // 在庫払出数量
		StockUnitPrice    float64   // 在庫払出単価
		StockCost         float64   // 在庫払出金額
		RecordDate        time.Time // 登録日
		OrderDate         time.Time // 発注日
		DeliveryDate      time.Time // 納期
		ReplyDeliveryDate time.Time // 回答納期
		RealDeliveryDate  time.Time // 納入日
		OrderDivision     string    // 発注区分
		Maker             string    // メーカ
		Material          string    // 材質
		Quantity          float64   // 員数
		OrderQuantity     float64   // 必要数
		OrderNum          float64   // 部品部品発注数
		OrderRest         float64   // 発注残数
		OrderUnitPrice    float64   // 発注単価
		OrderCost         float64   // 発注金額
		ProgressLevel     string    // 進捗レベル
		Process           string    // 工程名
		Vendor            string    // 仕入先略称
		OrderNo           int16     // オーダーNo
		DeliveryPlace     string    // 納入場所名
		Misc              string    // 部品備考
		CostCode          int16     // 原価費目ｺｰﾄﾞ
		CostName          string    // 原価費目名
	}

	/*
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
			"仕入原価単価" INTEGER,
			"仕入原価金額" INTEGER,
			"在庫払出数量" TEXT,
			"在庫払出単価" INTEGER,
			"在庫払出金額" INTEGER,
			"登録日" DATE,
			"発注日" DATE,
			"納期" DATE,
			"回答納期" DATE,
			"納入日" DATE,
			"発注区分" TEXT,
			"メーカ" TEXT,
			"材質" TEXT,
			"員数" TEXT,
			"必要数" TEXT,
			"部品発注数" TEXT,
			"発注残数" TEXT,
			"発注単価" REAL,
			"発注金額" REAL,
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
	db, err := sql.Open("sqlite3", FILENAME)
	if err != nil {
		log.Fatal(err)
	}
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	qf = qframe.ReadSQL(tx, qsql.Query(SQLQ), qsql.SQLite())
	log.Println("qframe:", qf)
}

func main() {
	// Router
	r := gin.Default()
	r.Static("/static", "./static")
	r.LoadHTMLGlob("template/*.tmpl")

	// API
	r.GET("/", func(c *gin.Context) {
		table := Frame2Table(qf)
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
				"msg": "Bad Query",
			})
			return
		}
		log.Println(fmt.Sprintf("query: %#v", q))

		// Search keyword by query parameter
		filtered := q.search()
		table := Frame2Table(filtered)
		if len(table) == 0 {
			c.HTML(http.StatusBadRequest, "table.tmpl", gin.H{
				"msg": "検索結果がありません",
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
	stringView := map[string]qframe.StringView{
		"ユニットNo": qf.MustStringView("ユニットNo"),
		"品番":     qf.MustStringView("品番"),
		"品名":     qf.MustStringView("品名"),
		"形式寸法":   qf.MustStringView("形式寸法"),
		"メーカ":    qf.MustStringView("メーカ"),
		"材質":     qf.MustStringView("材質"),
		"工程名":    qf.MustStringView("工程名"),
		"納入場所名":  qf.MustStringView("納入場所名"),
	}
	// intView := map[string]qframe.IntView{
	// 	"必要数": qf.MustIntView("必要数"),
	// 	"発注数": qf.MustIntView("部品発注数"),
	// }
	// floatview := map[string]qframe.FloatView{
	// 	"発注単価": qf.MustFloatView("発注単価"),
	// 	"発注金額": qf.MustFloatView("発注金額"),
	// }

	// slices := [][]interface{}{}
	slices := map[string]Column{}
	for k := range stringView {
		v := qf.MustStringView(k)
		slices[k] = toSlice(v)
	}

	slicesf := map[string]Columnf{
		"発注単価": qf.MustFloatView("発注単価").Slice(),
		"発注金額": qf.MustFloatView("発注金額").Slice(),
	}

	// 	toSlice(view["品番"]),
	// 	toSlice(view["品名"]),
	// 	toSlice(view["形式寸法"]),
	// }

	// NameとTypeは常に表示する仕様
	for i := 0; i < len(slices["品名"]); i++ {
		if i >= MAXROW { // 最大1000件表示
			break
		}
		r := Row{
			UnitNo:        slices["ユニットNo"][i],
			Pid:           slices["品番"][i],
			Name:          slices["品名"][i],
			Type:          slices["形式寸法"][i],
			Maker:         slices["メーカ"][i],
			Material:      slices["材質"][i],
			Process:       slices["工程名"][i],
			DeliveryPlace: slices["納入場所名"][i],

			// Float
			OrderRest:      slicesf["発注単価"][i],
			OrderUnitPrice: slicesf["発注金額"][i],
		}
		table = append(table, r)
	}
	return
}
