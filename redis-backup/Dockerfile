FROM google/golang

WORKDIR /gopath/src/app
ADD . /gopath/src/app/
RUN \
  go get code.google.com/p/goauth2/oauth && \
  go get code.google.com/p/google-api-go-client/storage/v1 && \
  go get app

CMD []
ENTRYPOINT ["/gopath/bin/app"]
