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
	FILENAME = "../../data/sqlite3.db"
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
			`
	// WHERE rowid > 800000

	// MAXROW : qfからTableへ変換する最大行数
	MAXROW = 1000
)

var (
	showVersion bool
	qf          qframe.QFrame
)

type (
	// Table : HTMLへ書き込むための行指向のstruct
	Table []Row
	// Row : Tableの一行
	Row struct {
		UnitNo string
		Pid    string
		Name   string
		Type   string
	}
	// Query : URLクエリパラメータ
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
		fmt.Printf("query: %#v\n", q)

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
			"msg":   fmt.Sprintf("%#v を検索, %d件を表示", q, len(table)),
			"table": table,
		})
	})

	r.Run()
}

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
		if i >= MAXROW { // 最大1000件表示
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
