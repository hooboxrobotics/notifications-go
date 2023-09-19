FROM golang:latest

WORKDIR /app

COPY . .

RUN go mod download

RUN go env -w GOFLAGS="-buildvcs=false"

RUN go install github.com/cosmtrek/air@latest

RUN go build -o main -buildvcs=false

EXPOSE 3000

CMD ["air -c .air.toml"]