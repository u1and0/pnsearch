install:
	go build -o ../../pnsearch
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CXX=x86_64-w64-mingw32-g++ CC=x86_64-w64-mingw32-gcc go build -o ../../pnsearch.exe
	go run main.go -v > ../../VERSION.txt
	rsync -auv --delete template ../..
