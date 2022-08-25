package main

import (
	"database/sql"
	"embed"
	"errors"
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
	VERSION = "v0.3.1r"
	// FILENAME : sqlite3 database file
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
	/*コマンドフラグ*/
	showVersion bool
	debug       bool
	portnum     int
	filename    string
	// allData : SQLQの実行でメモリ内に読み込んだ全データ
	allData qframe.QFrame
	/*template以下の全てのファイルをバイナリへ取り込み*/
	//go:embed template/*
	f embed.FS
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
		log.Panicln(err)
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
		log.Panicln(err)
	}
	tx, err := db.Begin()
	if err != nil {
		log.Panicln(err)
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
		c.HTML(http.StatusOK, "noui.tmpl", gin.H{
			"msg":    fmt.Sprintf("テストページ / トップから%d件を表示", len(table)),
			"table":  table,
			"header": FieldNameToAlias(allData.ColumnNames()),
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
			c.JSON(http.StatusBadRequest, gin.H{"msg": msg, "query": q})
		}
		return
	}
	log.Printf("query: %#v", q)

	// Empty query
	if reflect.DeepEqual(q, newQuery()) {
		msg := "検索キーワードがありません"
		if templateName != "" {
			c.HTML(http.StatusBadRequest, templateName, gin.H{"msg": msg, "query": q})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"msg": msg, "query": q})
		}
		return
	}

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
	if q.SortOrder != "" { // Default descending order
		qf = qf.Sort(qframe.Order{Column: q.SortOrder, Reverse: !q.SortAsc})
		if debug {
			log.Println("Sorted QFrame\n", qf)
		}
	}

	var (
		jsonObj []Object
		err     error
	)
	if templateName == "" {
		jsonObj, err = ToTable(qf).ToObject()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"msg": err, "query": q})
			return
		}
	}

	// 列選択Selectだけ表示。 列選択Selectがない場合はすべての列を表示。
	if len(q.Select) != 0 {
		cols := AliasToFieldName(q.Select)
		qf = qf.Select(cols...)
	}
	if debug {
		log.Println("Selected QFrame\n", qf)
	}
	l := qf.Len()
	table := ToTable(qf)
	if templateName != "" { // return HTML template
		c.HTML(http.StatusOK, templateName, gin.H{
			"msg":      fmt.Sprintf("検索結果: %d件中%d件を表示", l, len(table)),
			"query":    q,
			"header":   FieldNameToAlias(qf.ColumnNames()),
			"table":    table,
			"sortable": []string{"製番", "登録日", "発注日", "納期", "回答納期", "納入日"},
			"labels":   LabelMaker(allData.ColumnNames()),
		})
	} else { // return JSON
		msg := fmt.Sprintf("%#v を検索, %d件を表示", q, l)
		c.IndentedJSON(http.StatusOK, gin.H{
			"msg":    msg,
			"query":  q,
			"length": l,
			"table":  jsonObj,
		})
	}
}

/*クエリパラメータ関連*/

type (
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
	// Option : ソートオプション、AND検索OR検索切り替え
	Option struct {
		SortOrder string `form:"orderby"`
		SortAsc   bool   `form:"asc"`
		OR        bool   `form:"or"`
	}
)

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

/*UIラベル, フィールド名変換API関連*/

var (
	spellMap = bimap.NewBiMapFromMap(
		map[string]string{
			// フィールド名: 表示名
			"製番_品名":  "製番名称",
			"ユニットNo": "要求番号",
			"員数":     "数量",
			"形式寸法":   "型式",
			"材質":     "装置名",
		})
)

type (
	// Labels : ラベル
	// 順序保持のためにmapではなくあえてslice of structを使っている
	Labels []Label
	// Label : ラベル
	// Alias(表示名), Name(SQLデータのカラム名)の組み合わせ
	Label struct{ Alias, Name string }
)

// LabelMaker : Labelsを与えられた表示名sliceから作る
func LabelMaker(names []string) Labels {
	labels := make(Labels, len(names))
	for i, l := range FieldNameToAlias(names) {
		labels[i] = Label{Alias: l, Name: names[i]}
	}
	return labels
}

// FieldNameToAlias : SQLデータベースカラム名(データ名)をHTMLテーブルヘッダー名(表示名)へ変換する
func FieldNameToAlias(bfr []string) []string {
	var aft = make([]string, len(bfr))
	for i, k := range bfr {
		if v, ok := spellMap.Get(k); ok {
			aft[i] = v
		} else {
			aft[i] = k
		}
	}
	return aft
}

// AliasToFieldName : HTMLテーブルヘッダー名(表示名)をSQLデータベースカラム名(データ名)へ変換する
func AliasToFieldName(bfr []string) []string {
	var aft = make([]string, len(bfr))
	for i, v := range bfr {
		if k, ok := spellMap.GetInverse(v); ok {
			aft[i] = k
		} else {
			aft[i] = v
		}
	}
	return aft
}

/*Table, JSONオブジェクトAPI関連*/

const (
	/* テーブル情報
	検索、ソートのことは考えず
	表示とコーディングしやすさのことを考慮して、
	すべてTEXT型に変更した。
					CREATE TABLE order2 (
					"index" INTEGER,*/
	受注No     = iota // "受注No" TEXT,
	製番              // "製番" TEXT,
	製番名称            // "製番_品名" TEXT,
	要求番号            // "ユニットNo" TEXT,
	品番              // "品番" TEXT,
	品名              // "品名" TEXT,
	型式              // "形式寸法" TEXT,
	単位              // "単位" TEXT,
	仕入原価数量          // "仕入原価数量" TEXT,
	仕入原価単価          // "仕入原価単価" TEXT,
	仕入原価金額          // "仕入原価金額" TEXT,
	在庫払出数量          // "在庫払出数量" TEXT,
	在庫払出単価          // "在庫払出単価" TEXT,
	在庫払出金額          // "在庫払出金額" TEXT,
	登録日             // "登録日" TEXT,
	発注日             // "発注日" TEXT,
	納期              // "納期" TEXT,
	回答納期            // "回答納期" TEXT,
	納入日             // "納入日" TEXT,
	発注区分            // "発注区分" TEXT,
	メーカ             // "メーカ" TEXT,
	材質              // "材質" TEXT,
	数量              // "員数" TEXT,
	必要数             // "必要数" TEXT,
	部品発注数           // "部品発注数" TEXT,
	発注残数            // "発注残数" TEXT,
	発注単価            // "発注単価" TEXT,
	発注金額            // "発注金額" TEXT,
	進捗レベル           // "進捗レベル" TEXT,
	工程名             // "工程名" TEXT,
	仕入先略称           // "仕入先略称" TEXT,
	オーダーNo          // "オーダーNo" TEXT,
	納入場所名           // "納入場所名" TEXT,
	部品備考            // "部品備考" TEXT,
	原価費目ｺｰﾄﾞ        // "原価費目ｺｰﾄﾞ" TEXT,
	原価費目名           // "原価費目名" TEXT
	/*);*/
)

type (
	// Table : HTMLへ書き込むための行指向の構造体
	Table []Column
	// Column : toSlice()で変換されるqfの列
	Column []string

	// Object : JSONオブジェクト返すための列試行の構造体
	Object struct {
		ReceivedOrderNo   string `json:"受注No"`
		ProductNo         string `json:"製番"`
		ProductNoName     string `json:"製番名称"`
		UnitNo            string `json:"要求番号"`
		Pid               string `json:"品番"`
		Name              string `json:"品名"`
		Type              string `json:"型式"`
		Unit              string `json:"単位"`
		PurchaseQuantity  string `json:"仕入原価数量"`
		PurchaseUnitPrice string `json:"仕入原価単価"`
		PurchaseCost      string `json:"仕入原価金額"`
		StockQuantity     string `json:"在庫払出数量"`
		StockUnitPrice    string `json:"在庫払出単価"`
		StockCost         string `json:"在庫払出金額"`
		RecordDate        string `json:"登録日"`
		OrderDate         string `json:"発注日"`
		DeliveryDate      string `json:"納期"`
		ReplyDeliveryDate string `json:"回答納期"`
		RealDeliveryDate  string `json:"納入日"`
		OrderDivision     string `json:"発注区分"`
		Maker             string `json:"メーカ"`
		Material          string `json:"材質"`
		Quantity          string `json:"数量"`
		OrderQuantity     string `json:"必要数"`
		OrderNum          string `json:"部品部品発注数"`
		OrderRest         string `json:"発注残数"`
		OrderUnitPrice    string `json:"発注単価"`
		OrderCost         string `json:"発注金額"`
		ProgressLevel     string `json:"進捗レベル"`
		Process           string `json:"工程名"`
		Vendor            string `json:"仕入先略称"`
		OrderNo           string `json:"オーダーNo"`
		DeliveryPlace     string `json:"納入場所名"`
		Misc              string `json:"部品備考"`
		CostCode          string `json:"原価費目ｺｰﾄﾞ"`
		CostName          string `json:"原価費目名"`
	}
)

// ToObject : QFrame をJSONオブジェクトへ変換
func (table Table) ToObject() ([]Object, error) {
	o := make([]Object, len(table))
	fl := len(table.T())
	if fl != 36 {
		return o, errors.New(fmt.Sprintf("error: length mismatch %d, must 36", fl))
	}
	for i, r := range table {
		o[i].ReceivedOrderNo = r[受注No]
		o[i].ProductNo = r[製番]
		o[i].ProductNoName = r[製番名称]
		o[i].UnitNo = r[要求番号]
		o[i].Pid = r[品番]
		o[i].Name = r[品名]
		o[i].Type = r[型式]
		o[i].Unit = r[単位]
		o[i].PurchaseQuantity = r[仕入原価数量]
		o[i].PurchaseUnitPrice = r[仕入原価単価]
		o[i].PurchaseCost = r[仕入原価金額]
		o[i].StockQuantity = r[在庫払出数量]
		o[i].StockUnitPrice = r[在庫払出単価]
		o[i].StockCost = r[在庫払出金額]
		o[i].RecordDate = r[登録日]
		o[i].OrderDate = r[発注日]
		o[i].DeliveryDate = r[納期]
		o[i].ReplyDeliveryDate = r[回答納期]
		o[i].RealDeliveryDate = r[納入日]
		o[i].OrderDivision = r[発注区分]
		o[i].Maker = r[メーカ]
		o[i].Material = r[材質]
		o[i].Quantity = r[数量]
		o[i].OrderQuantity = r[必要数]
		o[i].OrderNum = r[部品発注数]
		o[i].OrderRest = r[発注残数]
		o[i].OrderUnitPrice = r[発注単価]
		o[i].OrderCost = r[発注金額]
		o[i].ProgressLevel = r[進捗レベル]
		o[i].Process = r[工程名]
		o[i].Vendor = r[仕入先略称]
		o[i].OrderNo = r[オーダーNo]
		o[i].DeliveryPlace = r[納入場所名]
		o[i].Misc = r[部品備考]
		o[i].CostCode = r[原価費目ｺｰﾄﾞ]
		o[i].CostName = r[原価費目名]
	}
	return o, nil
}

// ToTable : QFrame をTableへ変換
func ToTable(qf qframe.QFrame) Table {
	l := len(qf.ColumnNames())
	table := make(Table, l)
	for i, colName := range qf.ColumnNames() {
		table[i] = toSlice(qf, colName)
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
