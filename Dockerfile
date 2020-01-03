FROM golang:1.13.2 AS BUILD

RUN apt-get update && apt-get install -y libgeos-dev

RUN mkdir /elasticblast
WORKDIR /elasticblast

ADD go.mod .
ADD go.sum .
RUN go mod download

#now build source code
ADD . ./
RUN go build -o /go/bin/elasticblast



FROM golang:1.13.2

EXPOSE 9200

ENV BLAST_URL       ''

COPY --from=BUILD /go/bin/* /bin/
ADD startup.sh /

CMD [ "/startup.sh" ]
