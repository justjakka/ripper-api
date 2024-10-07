FROM golang:1.23

ADD . /go/src/ripper-api
WORKDIR /go/src/ripper-api

COPY keys /
COPY config.toml /

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /ripper-api

CMD ["/ripper-api", "serve"]