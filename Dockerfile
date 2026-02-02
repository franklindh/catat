FROM golang:1.24-alpine AS builder
WORKDIR /app


COPY go.mod go.sum ./
RUN go mod download


COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main main.go


RUN apk add --no-cache curl
RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.19.1/migrate.linux-amd64.tar.gz | tar xvz


FROM alpine:3.23
RUN apk add --no-cache netcat-openbsd
WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /app/migrate ./migrate
COPY start.sh .
COPY wait-for.sh .
COPY db/migration ./db/migration
RUN chmod +x /app/start.sh /app/wait-for.sh

EXPOSE 3000
CMD ["/app/main"]
ENTRYPOINT ["/app/start.sh"]