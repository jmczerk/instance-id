FROM golang:1.21

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
COPY utils/*.go ./utils/

RUN CGO_ENABLED=0 GOOS=linux go build -o /instance-id

CMD ["/instance-id"]
