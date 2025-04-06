FROM golang:1.24-alpine AS builder
COPY ../internal /internal
WORKDIR /app
COPY ./location-history-management/ .
RUN go mod download
RUN go test -v ./...
RUN env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./location-history-management .

FROM alpine:3.21.3
WORKDIR /app
COPY --from=builder /app/location-history-management .
EXPOSE 8080
EXPOSE 50051
CMD ["./location-history-management"]