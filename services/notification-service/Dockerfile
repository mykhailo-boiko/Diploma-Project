FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /notification-service ./services/notification-service/cmd/

FROM alpine:3.21
RUN apk --no-cache add ca-certificates
COPY --from=builder /notification-service /notification-service
EXPOSE 8006
CMD ["/notification-service"]
