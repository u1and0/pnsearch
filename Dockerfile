# Usage:
# ```
# $ docker run -d --rm u1and0/pnsearch [pnsearch OPTIONS]...
# ```
#
# Usage of pnsearch:
#   -debug
#     	Run debug mode
#   -f string
#     	SQL database file path (default "./data/sqlite3.db")
#   -p int
#     	Access port (default 9000)
#   -v	Show version
# ```

FROM golang:1.19.0-alpine3.16 AS go_builder
WORKDIR /work
ENV CGO_ENABLED=1 \
    GOOS=linux \
    GOARCH=amd64
RUN apk update && apk add build-base
COPY ./go.mod /work/go.mod
COPY ./go.sum /work/go.sum
COPY ./main.go /work/main.go
COPY ./template /work/template
RUN go build -a -ldflags '-linkmode external -extldflags "-static"'

FROM alpine:3.16.2
COPY --from=go_builder /work/pnsearch /usr/bin/pnsearch
ENTRYPOINT ["/usr/bin/pnsearch"]

LABEL maintainer="u1and0 <e01.ando60@gmail.com>"\
      description="Run PN Search Web Server"\
      version="v0.3.1"
