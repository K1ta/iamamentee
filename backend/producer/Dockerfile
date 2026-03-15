FROM golang:1.26-alpine

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY producer.go .
RUN go mod download
RUN go build -o producer producer.go

CMD ["./producer"]