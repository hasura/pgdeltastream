FROM golang:1.10

ADD . /go/src/github.com/hasura/pgdeltastream

#RUN go get -u github.com/golang/dep/cmd/dep
#RUN dep ensure

# workaround since the vendored version using dep doesn't compile
RUN go get -u github.com/gin-gonic/gin
RUN go get -u github.com/jackc/pgx

WORKDIR /go/src/github.com/hasura/pgdeltastream
RUN go build .
EXPOSE 12312
CMD [ "./pgdeltastream", "postgres", "localhost", "localhost:12312" ]
