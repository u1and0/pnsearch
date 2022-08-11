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
	// PORT : default port num
	PORT = 9000
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
	// Table : HTMLへ書き込むための行指向の構造体
	Table []Row
	// Object : JSONオブジェクト返すための列試行の構造体
	Object struct {
		UnitNo           Column `json:"ユニットNo"`
		Pid              Column `json:"品番"`
		Name             Column `json:"品名"`
		Type             Column `json:"形式寸法"`
		Maker            Column `json:"メーカ"`
		Material         Column `json:"材質"`
		Process          Column `json:"工程名"`
		DeliveryPlace    Column `json:"納入場所名"`
		OrderUnitPrice   Column `json:"発注単価"`
		OrderCost        Column `json:"発注金額"`
		OrderDate        Column `json:"発注日"`
		RealDeliveryDate Column `json:"納入日"`
	}
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
		SortOrder string `form:"sort"`
		SortAsc   bool   `form:"asc"`
	}
)

// Show version
func init() {
	flag.BoolVar(&showVersion, "v", false, "Show version")
	flag.BoolVar(&debug, "debug", false, "Run debug mode")
	flag.IntVar(&portnum, "p", PORT, "Access port")
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

	s := r.Group("/search")
	{
		s.GET("/", func(c *gin.Context) { ReturnTempl(c, "table.tmpl") })
		s.GET("/ui", func(c *gin.Context) { ReturnTempl(c, "ui.tmpl") })
		s.GET("/json", func(c *gin.Context) { ReturnTempl(c, "") })
	}

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

	// 原因不明だがfunctionや配列でregexp.MustCompile()してもうまく検索されないので
	// スライスで冗長ながら書き下すしかない。
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

// ReturnTempl : HTMLテンプレートを返す。
// テンプレート名がない場合はJSONを返す。
func ReturnTempl(c *gin.Context, templateName string) {
	// Extract query
	q := new(Query)
	if err := c.ShouldBind(q); err != nil {
		msg := fmt.Sprintf("%#v Bad Query", q)
		if templateName != "" {
			c.HTML(http.StatusBadRequest, templateName, gin.H{"msg": msg, "query": q})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"msg": msg, "query": q})
		}
		return
	}
	log.Println(fmt.Sprintf("query: %#v", q))

	// Search keyword by query parameter
	filtered := q.search()
	if filtered.Len() == 0 {
		msg := "検索結果がありません"
		if templateName != "" {
			c.HTML(http.StatusBadRequest, templateName, gin.H{"msg": msg, "query": q})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"msg": msg, "query": q})
		}
		return
	}

	// Display result
	// Default descending order
	sorted := filtered.Sort(qframe.Order{Column: q.SortOrder, Reverse: !q.SortAsc})
	l := filtered.Len()
	if templateName != "" {
		table := T(sorted)
		msg := fmt.Sprintf("検索結果: %d件中%d件を表示", l, len(table))
		c.HTML(http.StatusOK, templateName, gin.H{"msg": msg, "table": table, "query": q})
	} else {
		msg := fmt.Sprintf("%#v を検索, %d件を表示", q, l)
		jsonObj := J(sorted)
		c.IndentedJSON(http.StatusOK, gin.H{"msg": msg, "length": l, "table": jsonObj, "query": q})
	}
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

// J : QFrame をJSONオブジェクトへ変換
func J(qf qframe.QFrame) (obj Object) {
	obj.UnitNo = toSlice(qf, "ユニットNo")
	obj.Pid = toSlice(qf, "品番")
	obj.Name = toSlice(qf, "品名")
	obj.Type = toSlice(qf, "形式寸法")
	obj.Maker = toSlice(qf, "メーカ")
	obj.Material = toSlice(qf, "材質")
	obj.Process = toSlice(qf, "工程名")
	obj.DeliveryPlace = toSlice(qf, "納入場所名")
	obj.OrderUnitPrice = toSlice(qf, "発注単価")
	obj.OrderCost = toSlice(qf, "発注金額")
	obj.OrderDate = toSlice(qf, "発注日")
	obj.RealDeliveryDate = toSlice(qf, "納入日")
	return
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
