FROM golang:1.16-alpine as builder

RUN apk --update upgrade && \
    apk add build-base gcc sqlite && \
    rm -rf /var/cache/apk/*

RUN mkdir -p $GOPATH/src/github.com/thiagozs/geolocation-go

COPY . $GOPATH/src/github.com/thiagozs/geolocation-go/

RUN cd $GOPATH/src/github.com/thiagozs/geolocation-go/; go mod tidy

RUN cd $GOPATH/src/github.com/thiagozs/geolocation-go/; go build -o $GOPATH/bin/geolocation 


FROM alpine:3.22.2

RUN apk --no-cache add ca-certificates

RUN apk --update upgrade && \
    apk add tzdata && \
    rm -rf /var/cache/apk/*

EXPOSE 5000

RUN mkdir -p /bin/db
COPY --from=builder /go/src/github.com/thiagozs/geolocation-go/db/GeoLite2-City.mmdb bin/db/GeoLite2-City.mmdb
COPY --from=builder /go/bin/geolocation /bin/geolocation

WORKDIR /bin

CMD ["geolocation", "runserver", "--http=5000"]