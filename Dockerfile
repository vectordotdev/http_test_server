FROM golang:1.13 AS builder

WORKDIR /home
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -mod vendor -a -installsuffix nocgo -o /http_test_server .

FROM scratch
COPY --from=builder /http_test_server ./
ENTRYPOINT ["./http_test_server"]
