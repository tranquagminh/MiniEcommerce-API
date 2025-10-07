FROM golang:1.25.1 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o user-service ./cmd/main.go

FROM gcr.io/distroless/static-debian11
COPY --from=builder /app/user-service /user-service
EXPOSE 8081
CMD ["/user-service"]