FROM alpine:3.5

ADD . /go/src/github.com/rlg2161/nsq-topic-cleanup

RUN apk update && apk add make gcc g++ libstdc++ icu-dev ncurses-dev git bash go curl && \
    cd /go/src/github.com/rlg2161/nsq-topic-cleanup && GOPATH=/go/ make

CMD "tail -f /dev/null"
