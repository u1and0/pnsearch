発注履歴データ検索ツール

See [Github - u1and0/pnsearch](https://github.com/u1and0/pnsearch)


# ユーザー向けの説明

| URL    | 説明 |
|-----------------------------------------|-------------------------------------|
| http://192.168.XXX.XXX:9000             | テストページトップ1000件を表示します。 |
| http://192.168.XXX.XXX:9000/search      | 要求票の友用、テーブルのみの表示 |
| http://192.168.XXX.XXX:9000/search/ui   | 検索UI付き、テーブル表示。普通のユーザーはこのページだけを使います。 |
| http://192.168.XXX.XXX:9000/search/csv  | CSV形式表示、エクスポートできます。 |
| http://192.168.XXX.XXX:9000/search/json | JSON形式で表示、エクスポートできます。 |

品番にAA を含む行
	http://192.168.XXX.XXX:9000/search?品番=AA

品番にAA かつ 品名にリングを含む行
	http://192.168.XXX.XXX:9000/search?品番=AA&品名=リング

品番にAA かつ 品名にリング　かつ 形式寸法にSW　かつ　要求番号に TBD 73か74か75か76を含む行
	http://192.168.XXX.XXX:9000/search?品番=AA&品名=リング&形式寸法=SW&要求番号=TBD 7[3-6]

スペースは任意の数の文字列(検索に正規表現を使用)に変換されます。


# 運用者向けの説明

下記exeを実行すると自分のPCをサーバーとして使えます。（ポート開放が必要かもしれません。）
上記例のIPアドレスを自分のものに変えるか、"localhost"に変えてください。
	...¥PNsearch¥pnsearch.exe

下記dbファイルのテーブル名"order2"を参照しています。
	...¥PNsearch¥data¥sqlite3.db

実行ファイルの使い方
```
$ pnsearch.exe -h
Usage of pnsearch:
  -debug
    	Run debug mode
  -f string
    	SQL database file path (default "./data/sqlite3.db")
  -p int
    	Access port (default 9000)
  -v	Show version
```


# 開発者向けの説明

ソースコード: main.go
	...¥PNsearch¥src¥pnsearch¥main.go

HTMLテンプレート: 表示を変えられます。
	...¥PNsearch¥template¥table.tmpl

## ビルド, インストール
gccライブラリが必要です。

```bash
# Alpine Linux
$ apk update && apk add build-base
# Debian/Ubuntu
$ apt-get update && apt-get install build-essential
# Archlinux
$ pacman -Syu mingw-w64-gcc mingw-w64-binutils gcc-multilib
```

### Linux

```bash
$ go build
```

上記のような単純なビルドの場合は、実行環境にもgccライブラリが必要です。
また、適宜オプションが必要です。

```bash
$ GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build
```

ワンバイナリで実行できるようにビルドする場合はオプションが必要です。

```bash
$ GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -a -ldflags '-linkmode external -extldflags "-static"'
```

### Windows

```bash
$ GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CXX=x86_64-w64-mingw32-g++ CC=x86_64-w64-mingw32-gcc go build -o pnsearch.exe
```

### Docker

```bash
$ docker pull u1and0/pnsearch
```

See [docker hub - u1and0/pnsearch](https://hub.docker.com/repository/docker/u1and0/pnsearch)

```bash
$ docker run -t --rm -v /path/to/data:/data -p 9000:9000 u1and0/pnsearch
```
