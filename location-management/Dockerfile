FROM golang:1.24-alpine AS builder
COPY ../internal /internal
RUN mkdir -p /location-history-management/proto
COPY ../location-history-management/proto /location-history-management/proto
WORKDIR /app
COPY ./location-management/ .
RUN go mod download
RUN go test -v ./...
RUN env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./location-management .

FROM alpine:3.21.3
WORKDIR /app
COPY --from=builder /app/location-management .
EXPOSE 8080
CMD ["./location-management"]