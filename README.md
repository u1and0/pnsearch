# PN Search
発注履歴データ検索ツール


## ユーザー向けの説明

| URL    | 説明 |
|-----------------------------------------|-------------------------------------|
| http://192.168.XXX.XXX:9000              | テストページトップ1000件を表示します。 |
| http://192.168.XXX.XXX:9000/search       | 要求票の友用、テーブルのみの表示 |
| http://192.168.XXX.XXX:9000/search/ui    | 検索UI付き、テーブル表示。普通のユーザーはこのページだけを使います。 |
| http://192.168.XXX.XXX:9000/search/json　| JSON形式で表示、エクスポートできます。 |
| http://192.168.XXX.XXX:9000/search/csv　 | *未実装* CSV形式表示、エクスポートできます。 |

品番にAA を含む行
	http://192.168.XXX.XXX:9000/search?品番=AA

品番にAA かつ 品名にリングを含む行
	http://192.168.XXX.XXX:9000/search?品番=AA&品名=リング

品番にAA かつ 品名にリング　かつ 形式寸法にSW　かつ　要求番号に TBD 73か74か75か76を含む行
	http://192.168.XXX.XXX:9000/search?品番=AA&品名=リング&形式寸法=SW&要求番号=TBD 7[3-6]

スペースは任意の数の文字列(検索に正規表現を使用)に変換されます。


## 運用者向けの説明

下記exeを実行すると自分のPCをサーバーとして使えます。（ポート開放が必要かもしれません。）
上記例のIPアドレスを自分のものに変えるか、"localhost"に変えてください。
	...¥PNsearch¥pnsearch.exe

下記dbファイルのテーブル名"order2"を参照しています。
	...¥PNsearch¥data¥sqlite3.db

実行ファイルの使い方
$ pnsearch.exe -h
Usage of ../../pnsearch:
  -debug
        Run debug mode
  -f string
        SQL database file path (default "./data/sqlite3.db")
  -p int
        Access port (default 9000)
  -v    Show version


## 開発者向けの説明

ソースコード: main.go
	...¥PNsearch¥src¥pnsearch¥main.go

HTMLテンプレート: 表示を変えられます。
	...¥PNsearch¥template¥table.tmpl

### ビルド, インストール
gccライブラリが必要です。

```bash
$ pacman -Syu mingw-w64-gcc mingw-w64-binutils gcc-multilib
```

#### Linux
`go build`

#### Windows
`GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CXX=x86_64-w64-mingw32-g++ CC=x86_64-w64-mingw32-gcc go build -o pnsearch.exe`
