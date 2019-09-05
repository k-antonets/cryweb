FROM golang:alpine

RUN mkdir /src && mkdir -p /app

COPY ./ /src/

RUN go build -o /app/server /src/

ENTRYPOINT ["/app/server"]