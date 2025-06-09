FROM golang:1.24-alpine3.22 AS build
RUN apk --no-cache add gcc g++ make ca-certificates
WORKDIR /go/src/Microservices-based-E-commerce-System
COPY go.mod go.sum ./
COPY catalog catalog
RUN GO111MODULE=on go build -o /go/bin/catalog ./catalog/cmd/catalog
FROM alpine:3.18
RUN addgroup -S app && adduser -S app -G app
WORKDIR /usr/bin
COPY --from=build /go/bin/catalog .
USER app
EXPOSE 8080
CMD ["catalog"]