FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/service .

FROM scratch
COPY --from=builder /bin/service /service
EXPOSE 8080
ENTRYPOINT ["/service"]
