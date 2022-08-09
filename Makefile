install:
	GOOS=windows GOARCH=amd64 go build -o ../../pnsearch.exe
	go run main.go -v > ../../VERSION.txt
	rsync -auv --delete template ../..
