# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY . .

RUN go build -o favourites main.go

# Runtime stage
FROM alpine:3.22
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/favourites .

EXPOSE 8080
CMD ["./favourites"]
