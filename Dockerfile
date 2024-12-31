FROM golang:1.23.4-alpine

WORKDIR /usr/src/ripper-api
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /usr/bin/ripper-api

CMD ["ripper-api"]
