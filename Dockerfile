FROM golang:1.24.2

WORKDIR /app

COPY . .

RUN go mod tidy \
    && go build -o loadbalancer ./cmd/loadbalancer

EXPOSE 8080

CMD ["./loadbalancer", "-config", "configs/config.yaml"]
