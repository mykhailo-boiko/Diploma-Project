FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /logistics-service ./services/logistics-service/cmd/

FROM alpine:3.21
RUN apk --no-cache add ca-certificates
COPY --from=builder /logistics-service /logistics-service
EXPOSE 8004
CMD ["/logistics-service"]
