FROM golang:1.24-alpine3.22 AS build
RUN apk --no-cache add gcc g++ make ca-certificates
WORKDIR /go/src/Microservices-based-E-commerce-System
COPY go.mod go.sum ./
COPY account account
COPY catalog catalog
COPY order order
COPY graphql graphql
WORKDIR /go/src/Microservices-based-E-commerce-System/graphql
RUN go build -o /go/bin/app .
FROM alpine:3.11
WORKDIR /usr/bin
RUN apk --no-cache add ca-certificates
COPY --from=build /go/bin/app .
EXPOSE 8080
CMD ["./app"]
