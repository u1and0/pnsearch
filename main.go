package main

import (
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"html/template"
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
	"github.com/vishalkuo/bimap"
)

const (
	// VERSION : version info
	VERSION = "v0.3.0"
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
	// LABEL : 列選択チェックボックスラベル
)

var (
	showVersion bool
	debug       bool
	allData     qframe.QFrame
	portnum     int
	filename    string
	//go:embed template/*
	f        embed.FS
	spellMap bimap.BiMap[string, string]
)

type (
	// Table : HTMLへ書き込むための行指向の構造体
	Table []Column
	// Object : JSONオブジェクト返すための列試行の構造体
	Object struct {
		ReceivedOrderNo   Column `json:"受注No"`
		ProductNo         Column `json:"製番  "`
		ProductNoName     Column `json:"製番_ 品名"`
		UnitNo            Column `json:"ユニットNo"`
		Pid               Column `json:"品番  "`
		Name              Column `json:"品名  "`
		Type              Column `json:"型式"`
		Unit              Column `json:"単位  "`
		PurchaseQuantity  Column `json:"仕入原価数量"`
		PurchaseUnitPrice Column `json:"仕入原価単価"`
		PurchaseCost      Column `json:"仕入原価金額"`
		StockQuantity     Column `json:"在庫払出数量"`
		StockUnitPrice    Column `json:"在庫払出単価"`
		StockCost         Column `json:"在庫払出金額"`
		RecordDate        Column `json:"登録日"`
		OrderDate         Column `json:"発注日"`
		DeliveryDate      Column `json:"納期  "`
		ReplyDeliveryDate Column `json:"回答納期"`
		RealDeliveryDate  Column `json:"納入日"`
		OrderDivision     Column `json:"発注区分"`
		Maker             Column `json:"メーカ"`
		Material          Column `json:"材質  "`
		Quantity          Column `json:"員数  "`
		OrderQuantity     Column `json:"必要数"`
		OrderNum          Column `json:"部品部品発注数"`
		OrderRest         Column `json:"発注残数"`
		OrderUnitPrice    Column `json:"発注単価"`
		OrderCost         Column `json:"発注金額"`
		ProgressLevel     Column `json:"進捗レベル"`
		Process           Column `json:"工程名"`
		Vendor            Column `json:"仕入先略称"`
		OrderNo           Column `json:"オーダーNo"`
		DeliveryPlace     Column `json:"納入場所名"`
		Misc              Column `json:"部品備考"`
		CostCode          Column `json:"原価費目ｺｰﾄﾞ"`
		CostName          Column `json:"原価費目名"`
	}
	// Column : toSlice()で変換されるqfの列
	Column []string

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

	// Query : URLクエリパラメータ 検索キーワード
	Query struct {
		ProductNo string `form:"製番"`
		UnitNo    string `form:"要求番号"`
		Pid       string `form:"品番"`
		Name      string `form:"品名"`
		Type      string `form:"型式"`
		Maker     string `form:"メーカ"`
		Vendor    string `form:"仕入先"`
		Option
		Select []string `form:"select"`
	}
	// Labels : ラベル
	Labels []Label
	// Label : ラベル
	Label struct{ Name, Value string }
	// Option : ソートオプション、AND検索OR検索切り替え
	Option struct {
		SortOrder string `form:"orderby"`
		SortAsc   bool   `form:"asc"`
		OR        bool   `form:"or"`
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
	if _, err := os.Stat(filename); err != nil {
		log.Fatal(err)
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
	typemap := allData.ColumnTypeMap()
	// Drop NOT string type columns
	for k, v := range typemap {
		if v != "string" {
			allData = allData.Drop(k)
		}
	}
	log.Println("Loaded frame\n", allData)
}

// 変換BiMapの作成
func init() {
	maps := map[string]string{
		// フィールド名: 表示名
		"製番_品名":  "製番名称",
		"ユニットNo": "要求番号",
		"員数":     "数量",
		"形式寸法":   "型式",
		"材質":     "装置名",
	}
	spellMap = *bimap.NewBiMapFromMap(maps)
}

func main() {
	// Router
	r := gin.Default()
	templ := template.Must(template.New("").ParseFS(f, "template/*.tmpl"))
	r.SetHTMLTemplate(templ)

	// API
	r.GET("/", func(c *gin.Context) {
		table := ToTable(allData)
		if debug {
			log.Println(table)
		}
		header := ConvertHeader(&spellMap, allData.ColumnNames(), false)
		c.HTML(http.StatusOK, "noui.tmpl", gin.H{
			"msg":    fmt.Sprintf("テストページ / トップから%d件を表示", len(table)),
			"table":  table,
			"header": header,
		})
	})

	s := r.Group("/search")
	{
		s.GET("/", func(c *gin.Context) { ReturnTempl(c, "noui.tmpl") })
		s.GET("/ui", func(c *gin.Context) { ReturnTempl(c, "ui.tmpl") })
		s.GET("/json", func(c *gin.Context) { ReturnTempl(c, "") })
	}

	port := ":" + strconv.Itoa(portnum)
	r.Run(port)
}

func (q *Query) search() qframe.QFrame {
	// 原因不明だがfunctionや配列でregexp.MustCompile()してもうまく検索されないので
	// スライスで冗長ながら書き下すしかない。
	filters := []qframe.FilterClause{}
	// OR 検索にて、クエリが空文字の時
	// すべての文字列 ".*.*" を検索してしまうのを防ぐため
	// ifでfiltersにフィルターを追加するか条件節
	if q.ProductNo != "" {
		filters = append(filters, qframe.Filter{
			Comparator: func(p *string) bool {
				return regexp.MustCompile(ToRegex(q.ProductNo)).MatchString(toString(p))
			},
			Column: "製番",
		})
	}
	if q.UnitNo != "" {
		filters = append(filters, qframe.Filter{
			Comparator: func(p *string) bool {
				return regexp.MustCompile(ToRegex(q.UnitNo)).MatchString(toString(p))
			},
			Column: "ユニットNo",
		})
	}
	if q.Pid != "" {
		filters = append(filters, qframe.Filter{
			Comparator: func(p *string) bool {
				return regexp.MustCompile(ToRegex(q.Pid)).MatchString(toString(p))
			},
			Column: "品番",
		})
	}
	if q.Name != "" {
		filters = append(filters, qframe.Filter{
			Comparator: func(p *string) bool {
				return regexp.MustCompile(ToRegex(q.Name)).MatchString(toString(p))
			},
			Column: "品名",
		})
	}
	if q.Type != "" {
		filters = append(filters, qframe.Filter{
			Comparator: func(p *string) bool {
				return regexp.MustCompile(ToRegex(q.Type)).MatchString(toString(p))
			},
			Column: "形式寸法",
		})
	}
	if q.Maker != "" {
		filters = append(filters, qframe.Filter{
			Comparator: func(p *string) bool {
				return regexp.MustCompile(ToRegex(q.Maker)).MatchString(toString(p))
			},
			Column: "メーカ",
		})
	}
	if q.Vendor != "" {
		filters = append(filters, qframe.Filter{
			Comparator: func(p *string) bool {
				return regexp.MustCompile(ToRegex(q.Vendor)).MatchString(toString(p))
			},
			Column: "仕入先略称",
		})
	}
	if q.OR {
		return allData.Filter(qframe.Or(filters...))
	}
	return allData.Filter(qframe.And(filters...))
}

// ReturnTempl : HTMLテンプレートを返す。
// テンプレート名がない場合はJSONを返す。
func ReturnTempl(c *gin.Context, templateName string) {
	// Extract query
	q := newQuery()
	if err := c.ShouldBind(q); err != nil {
		msg := fmt.Sprintf("%#v Bad Query", q)
		if templateName != "" {
			c.HTML(http.StatusBadRequest, templateName, gin.H{"msg": msg, "query": fmt.Sprintf("%#v", q)})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"msg": msg, "query": fmt.Sprintf("%#v", q)})
		}
		return
	}
	log.Printf("query: %#v", q)

	// Search keyword by query parameter
	qf := q.search()
	if debug {
		log.Println("Filtered QFrame\n", qf)
	}

	// Search Failure
	if qf.Len() == 0 {
		msg := "検索結果がありません"
		if templateName != "" {
			c.HTML(http.StatusBadRequest, templateName, gin.H{"msg": msg, "query": q})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"msg": msg, "query": q})
		}
		return
	}

	// Search Success
	// Default descending order
	if q.SortOrder != "" {
		qf = qf.Sort(qframe.Order{Column: q.SortOrder, Reverse: !q.SortAsc})
		if debug {
			log.Println("Sorted QFrame\n", qf)
		}
	}
	if len(q.Select) != 0 {
		qf = qf.Select(q.Select...)
	}
	if debug {
		log.Println("Selected QFrame\n", qf)
	}
	l := qf.Len()
	if templateName != "" { // return HTML template
		var (

			// 順序保持のためにmapではなく[]structを使っている
			labels = Labels{
				// { 表示名, カラム名 }
				Label{"受注No", "受注No"},
				Label{"製番", "製番"},
				Label{"製番_品名", "製番_品名"},
				Label{"要求番号", "ユニットNo"},
				Label{"品番", "品番"},
				Label{"品名", "品名"},
				Label{"形式寸法", "形式寸法"},
				Label{"単位", "単位"},
				Label{"仕入原価数量", "仕入原価数量"},
				Label{"仕入原価単価", "仕入原価単価"},
				Label{"仕入原価金額", "仕入原価金額"},
				Label{"在庫払出数量", "在庫払出数量"},
				Label{"在庫払出単価", "在庫払出単価"},
				Label{"在庫払出金額", "在庫払出金額"},
				Label{"登録日", "登録日"},
				Label{"発注日", "発注日"},
				Label{"納期", "納期"},
				Label{"回答納期", "回答納期"},
				Label{"納入日", "納入日"},
				Label{"発注区分", "発注区分"},
				Label{"メーカ", "メーカ"},
				Label{"材質", "材質"},
				Label{"員数", "員数"},
				Label{"必要数", "必要数"},
				Label{"部品発注数", "部品発注数"},
				Label{"発注残数", "発注残数"},
				Label{"発注単価", "発注単価"},
				Label{"発注金額", "発注金額"},
				Label{"進捗レベル", "進捗レベル"},
				Label{"工程名", "工程名"},
				Label{"仕入先", "仕入先略称"},
				Label{"オーダーNo", "オーダーNo"},
				Label{"納入場所名", "納入場所名"},
				Label{"部品備考", "部品備考"},
				Label{"原価費目ｺｰﾄﾞ", "原価費目ｺｰﾄﾞ"},
				Label{"原価費目名", "原価費目名"},
			}

			sortable = []string{"製番", "登録日", "発注日", "納期", "回答納期", "納入日"}
			table    = ToTable(qf)
			msg      = fmt.Sprintf("検索結果: %d件中%d件を表示", l, len(table))
			header   = ConvertHeader(&spellMap, qf.ColumnNames(), false)
		)
		c.HTML(http.StatusOK, templateName, gin.H{
			"msg":      msg,
			"query":    q,
			"header":   header,
			"table":    table,
			"sortable": sortable,
			"labels":   labels,
		})
	} else { // return JSON
		msg := fmt.Sprintf("%#v を検索, %d件を表示", q, l)
		jsonObj := ToObject(qf)
		c.IndentedJSON(http.StatusOK, gin.H{
			"msg":    msg,
			"query":  q,
			"length": l,
			"table":  jsonObj,
		})
	}
}

// headerMap : SQLデータベースカラム名(データ名)をHTMLテーブルヘッダー名(表示名)へ変換する
func ConvertHeader(maps *bimap.BiMap[string, string], bfr []string, inverse bool) []string {
	var (
		aft = make([]string, len(bfr))
		ok  bool
		v   string
	)
	for i, k := range bfr {
		if !inverse {
			v, ok = maps.Get(k)
		} else {
			v, ok = maps.GetInverse(k)
		}
		if ok {
			aft[i] = v
		} else {
			aft[i] = k
		}
	}
	return aft
}

func toString(ptr *string) string {
	if (ptr == nil) || reflect.ValueOf(ptr).IsNil() {
		return ""
	}
	return *ptr
}

func toSlice(qf qframe.QFrame, colName string) (stringSlice []string) {
	view, err := qf.StringView(colName)
	if err != nil {
		log.Printf("No col %s", colName)
	}
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

// ToObject : QFrame をJSONオブジェクトへ変換
func ToObject(qf qframe.QFrame) (obj Object) {
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

// ToTable : QFrame をTableへ変換
func ToTable(qf qframe.QFrame) (table Table) {
	for _, colName := range qf.ColumnNames() {
		column := toSlice(qf, colName)
		table = append(table, column)
	}
	return table.T()
}

// T : transpose Table
func (table Table) T() Table {
	xl := len(table[0])
	xl = func() int { // table MAX length: MAXROW(1000)
		if MAXROW < xl {
			return MAXROW
		}
		return xl
	}()
	yl := len(table)
	result := make(Table, xl)
	for i := range result {
		result[i] = make([]string, yl)
	}
	for i := 0; i < xl; i++ {
		for j := 0; j < yl; j++ {
			result[i][j] = table[j][i]
		}
	}
	return result
}

func newQuery() *Query {
	o := Option{
		SortOrder: "発注日",
	}
	q := Query{
		Option: o,
		// Select: []string{"品番", "品名", "形式寸法"},
	}
	return &q
}
