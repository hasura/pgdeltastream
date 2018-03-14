FROM golang:1.10


ENV DBNAME "postgres"
ENV PGUSER "postgres"
ENV PGHOST "localhost"
ENV PGPORT 5432
ENV SERVERHOST "localhost"
ENV SERVERPORT 12312

ADD . /go/src/github.com/hasura/pgdeltastream

WORKDIR /go/src/github.com/hasura/pgdeltastream

RUN go get -u github.com/golang/dep/cmd/dep
RUN dep ensure --vendor-only

RUN go build .
EXPOSE ${SERVERPORT}
CMD  "./pgdeltastream" ${DBNAME} ${PGUSER} ${PGHOST} ${PGPORT} ${SERVERHOST} ${SERVERPORT} 
