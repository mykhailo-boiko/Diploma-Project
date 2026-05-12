FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /simulator-service ./services/simulator-service/cmd/

FROM alpine:3.21
RUN apk --no-cache add ca-certificates
COPY --from=builder /simulator-service /simulator-service
EXPOSE 8007
CMD ["/simulator-service"]
