FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod .
COPY *.go .
RUN go build -o aggregator .

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /app/aggregator .

EXPOSE 8081

CMD ["./aggregator"]
