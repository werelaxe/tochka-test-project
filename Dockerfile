FROM golang:1.11-alpine3.7


RUN apk update && apk upgrade && apk add git

COPY . /app
WORKDIR /app
RUN go get -d

CMD ["./run.sh"]

