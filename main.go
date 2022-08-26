package main

import (
	"bytes"
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
	VERSION = "v0.3.2r"
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
	q.SortOrder = "発注日"
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
			c.HTML(http.StatusBadRequest, templateName, gin.H{
				"msg":      msg,
				"query":    q,
				"sortable": []string{"製番", "登録日", "発注日", "納期", "回答納期", "納入日"},
				"labels":   LabelMaker(allData.ColumnNames()),
			})
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

	// 列選択Selectだけ表示。 列選択Selectがない場合はすべての列を表示。
	if len(q.Select) != 0 {
		cols := AliasToFieldName(q.Select)
		qf = qf.Select(cols...)
	}
	if debug {
		log.Println("Selected QFrame\n", qf)
	}
	if templateName != "" { // return HTML template
		l := qf.Len()
		table := ToTable(qf)
		c.HTML(http.StatusOK, templateName, gin.H{
			"msg":      fmt.Sprintf("検索結果: %d件中%d件を表示", l, len(table)),
			"query":    q,
			"header":   FieldNameToAlias(qf.ColumnNames()),
			"table":    table,
			"sortable": []string{"製番", "登録日", "発注日", "納期", "回答納期", "納入日"},
			"labels":   LabelMaker(allData.ColumnNames()),
		})
	} else { // return JSON
		var jsonObj bytes.Buffer
		if err := qf.ToJSON(&jsonObj); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"msg": err, "query": q})
			return
		}
		c.String(http.StatusOK, jsonObj.String())
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
		Filter
		Select []string `form:"select"`
	}
	// Option : ソートオプション、AND検索OR検索切り替え
	Option struct {
		SortOrder string `form:"orderby"`
		SortAsc   bool   `form:"asc"`
		OR        bool   `form:"or"`
	}
	Filter struct {
		Order    string `form:"発注"`
		Delivery string `form:"納入"`
	}
)

func newQuery() *Query {
	// o := Option{
	// 	SortOrder: "発注日",
	// }
	// q := Query{
	// 	Option: o,
	// 	// Select: []string{"品番", "品名", "形式寸法"},
	// }
	return &Query{}
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
				return regexp.MustCompile(q.ToRegex(q.ProductNo)).MatchString(toString(p))
			},
			Column: "製番",
		})
	}
	if q.UnitNo != "" {
		filters = append(filters, qframe.Filter{
			Comparator: func(p *string) bool {
				return regexp.MustCompile(q.ToRegex(q.UnitNo)).MatchString(toString(p))
			},
			Column: "ユニットNo",
		})
	}
	if q.Pid != "" {
		filters = append(filters, qframe.Filter{
			Comparator: func(p *string) bool {
				return regexp.MustCompile(q.ToRegex(q.Pid)).MatchString(toString(p))
			},
			Column: "品番",
		})
	}
	if q.Name != "" {
		filters = append(filters, qframe.Filter{
			Comparator: func(p *string) bool {
				return regexp.MustCompile(q.ToRegex(q.Name)).MatchString(toString(p))
			},
			Column: "品名",
		})
	}
	if q.Type != "" {
		filters = append(filters, qframe.Filter{
			Comparator: func(p *string) bool {
				return regexp.MustCompile(q.ToRegex(q.Type)).MatchString(toString(p))
			},
			Column: "形式寸法",
		})
	}
	if q.Maker != "" {
		filters = append(filters, qframe.Filter{
			Comparator: func(p *string) bool {
				return regexp.MustCompile(q.ToRegex(q.Maker)).MatchString(toString(p))
			},
			Column: "メーカ",
		})
	}
	if q.Vendor != "" {
		filters = append(filters, qframe.Filter{
			Comparator: func(p *string) bool {
				return regexp.MustCompile(q.ToRegex(q.Vendor)).MatchString(toString(p))
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
func (q *Query) ToRegex(s string) string {
	s = strings.ReplaceAll(s, "　", " ")  // 全角半角変換
	s = strings.ReplaceAll(s, "\t", " ") // タブ文字削除
	s = strings.TrimSpace(s)             // 左右の空白削除
	if q.OR {
		s = strings.Join(strings.Fields(s), `|`) // スペースを|に変換
		s = fmt.Sprintf(`(%s)`, s)
	} else {
		s = strings.Join(strings.Fields(s), `.*`) // スペースを.*に変換
		s = fmt.Sprintf(`.*%s.*`, s)
	}
	return `(?i)` + s // ignore case (?i)
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

type (
	// Table : HTMLへ書き込むための行指向の構造体
	Table []Column
	// Column : toSlice()で変換されるqfの列
	Column []string
)

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
