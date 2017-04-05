FROM alpine:3.5

ADD . /go/src/github.com/rlg2161/nsq-topic-cleanup

RUN apk update && apk add make gcc g++ libstdc++ icu-dev ncurses-dev git bash go curl && \
    cd /go/src/github.com/rlg2161/nsq-topic-cleanup && GOPATH=/go/ make && \
    chmod +x /go/src/github.com/rlg2161/nsq-topic-cleanup/runCleanup && \
    cp /go/src/github.com/rlg2161/nsq-topic-cleanup/runCleanup /etc/periodic/15min/runCleanup && \
    apk del make gcc g++ libstdc++ icu-dev ncurses-dev git go 

CMD crond -d 8 -f
