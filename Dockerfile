FROM golang:alpine

RUN mkdir /src && mkdir -p /app

COPY ./ /src/

WORKDIR /src/

RUN go build -o /app/server . && \
    cp -R templates /app && \
    cd / && rm -rf /src

WORKDIR /app/

ENTRYPOINT ["/app/server"]