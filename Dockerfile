FROM golang:1.25-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o go-tcp-lb main.go

# Second stage
FROM alpine:3.18

RUN apk add --no-cache bash

WORKDIR /app
COPY --from=build /app/go-tcp-lb .

EXPOSE 4000

ENTRYPOINT ["/app/go-tcp-lb"]