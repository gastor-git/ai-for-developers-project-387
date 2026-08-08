FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/backend .

FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates

COPY --from=builder /out/backend /usr/local/bin/backend

EXPOSE 8080

USER nobody

ENTRYPOINT ["/usr/local/bin/backend"]
