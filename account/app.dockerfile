FROM golang:1.24-alpine3.22 AS build
RUN apk --no-cache add gcc g++ make ca-certificates
WORKDIR /go/src/Microservices-based-E-commerce-System
COPY go.mod go.sum ./
COPY account account
RUN GO111MODULE=on go build -o /go/bin/account ./account/cmd/account
FROM alpine:3.18
RUN addgroup -S app && adduser -S app -G app
WORKDIR /usr/bin
COPY --from=build /go/bin/account .
USER app
EXPOSE 8080
CMD ["account"]