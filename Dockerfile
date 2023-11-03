FROM golang:1.19 as build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . ./

RUN go generate ./pkg/cmd/server.go
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o ./scanoss-semgrep-api ./cmd/server


FROM debian:buster-slim as production

WORKDIR /app
 
COPY --from=build /app/scanoss-semgrep-api /app/scanoss-semgrep-api

EXPOSE 5443

ENTRYPOINT ["./scanoss-semgrep-api"]
#CMD ["--help"]