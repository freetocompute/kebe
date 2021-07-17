FROM golang:1.16 AS builder

COPY . /source
WORKDIR /source
RUN go build -o /bin/kebe-login bin/login/main.go

FROM golang:1.16
WORKDIR /bin
COPY --from=builder /bin/kebe-login .
EXPOSE 8080

CMD ["/bin/kebe-login"]
