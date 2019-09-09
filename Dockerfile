FROM golang:alpine

RUN mkdir /src && mkdir -p /app

COPY ./ /src/

WORKDIR /src/

RUN go build -o /app/server .

ENTRYPOINT ["/app/server"]