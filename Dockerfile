FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/bootstrap ./cmd/bootstrap
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/node ./cmd/node

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /bin/bootstrap /bin/bootstrap
COPY --from=builder /bin/node /bin/node

EXPOSE 9099 50051 50052 50053 8080 8081 8082
ENTRYPOINT ["/bin/node"]
