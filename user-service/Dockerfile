FROM golang:1.25.1 AS builder
WORKDIR /app
COPY . .

RUN go mod tidy
RUN go build -o user-service ./cmd/main.go
COPY .env .env
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -a -installsuffix cgo -o user-service ./cmd/main.go

FROM gcr.io/distroless/static-debian11
COPY --from=builder /app/user-service /user-service
CMD ["/user-service"]