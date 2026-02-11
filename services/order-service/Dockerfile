FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /order-service ./services/order-service/cmd/

FROM alpine:3.21
RUN apk --no-cache add ca-certificates
COPY --from=builder /order-service /order-service
EXPOSE 8002
CMD ["/order-service"]
