FROM golang:1.10

ENV DBNAME "postgres"
ENV PGUSER "postgres"
# TODO: better handle default pass
ENV PGPASS "''" 
ENV PGHOST "localhost"
ENV PGPORT 5432
ENV SERVERHOST "localhost"
ENV SERVERPORT 12312

RUN go get -u github.com/golang/dep/cmd/dep

ADD . /go/src/github.com/hasura/pgdeltastream

WORKDIR /go/src/github.com/hasura/pgdeltastream

RUN dep ensure --vendor-only

RUN go build .

EXPOSE ${SERVERPORT}
CMD  "./pgdeltastream" "--db" ${DBNAME} "--user" ${PGUSER} "--password" ${PGPASS} "--pgHost" ${PGHOST} "--pgPort" ${PGPORT} "--serverHost" ${SERVERHOST} "--serverPort" ${SERVERPORT} 
