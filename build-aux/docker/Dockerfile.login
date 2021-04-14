FROM golang:buster

COPY . /source
WORKDIR /source
RUN go build -o /bin/kebe-login bin/login/main.go
WORKDIR /bin
EXPOSE 8080
