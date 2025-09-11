FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o loadtest-go

# Final image
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/loadtest-go .
EXPOSE 8080
CMD ["./loadtest-go", "server"]