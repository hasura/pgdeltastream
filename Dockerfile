FROM golang:1.10


ENV DBNAME "postgres"
ENV PGUSER "postgres"
ENV PGHOST "localhost"
ENV PGPORT 5432
ENV SERVERHOST "localhost"
ENV SERVERPORT 12312

ADD . /go/src/github.com/hasura/pgdeltastream

#RUN go get -u github.com/golang/dep/cmd/dep
#RUN dep ensure

# workaround since the vendored version using dep doesn't compile
RUN go get -u github.com/gin-gonic/gin
RUN go get -u github.com/jackc/pgx

WORKDIR /go/src/github.com/hasura/pgdeltastream
RUN go build .
EXPOSE ${SERVERPORT}
CMD [ "./pgdeltastream", ${DBNAME}, ${PGUSER}, ${PGHOST}, ${PGPORT}, ${SERVERHOST}, ${SERVERPORT} ]