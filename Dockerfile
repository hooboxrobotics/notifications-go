FROM golang:1.20.8-alpine3.18 as build

WORKDIR /app

COPY . .

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o main -buildvcs=false

FROM alpine:latest

WORKDIR /app

COPY --from=build /app/main /app

EXPOSE 3000

CMD ["./main"]