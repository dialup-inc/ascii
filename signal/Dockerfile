FROM golang:1.12

WORKDIR /signal
COPY . .

RUN go get -d -v .
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go install -v .

FROM alpine:latest

COPY --from=0 /go/bin/signal /signal

EXPOSE 8080

CMD ["/signal"]
