FROM golang:1 as builder
ARG CGO_ENABLED=0\
    GO_LDFLAGS="-s -w"
RUN mkdir -p /tmp/build
COPY . /tmp/build
WORKDIR /tmp/build/cmd/gitdir
RUN set -ex && ls -la && export &&\
    until go get -v ./...; do sleep 1; done &&\
    go build -o /tmp/gitdir -ldflags="$GO_LDFLAGS" -x -v &&\
    chmod +x /tmp/gitdir

FROM alpine:3 as runner
ENV GITDIR_BASE_DIR=/var/gitdir\
    GITDIR_BIND_ADDR=0.0.0.0:2222
RUN until apk add --no-cache git curl openssh-keygen ca-certificates; do sleep 1; done &&\
    update-ca-certificates
RUN adduser -D -H -s /bin/false gitdir gitdir
COPY --from=builder /tmp/gitdir /usr/bin/gitdir
RUN chmod +x /usr/bin/gitdir && ls -l /usr/bin/gitdir
RUN rm -rf /{tmp,var,opt}/*
RUN mkdir -p $GITDIR_BASE_DIR && chown -R gitdir:gitdir $GITDIR_BASE_DIR
RUN echo -e '#!/bin/sh\n \
( \
set -x; \
rm -rf $GITDIR_BASE_DIR/admin/unbared; \
until [ -d "$GITDIR_BASE_DIR/admin/admin.git" ]; do echo "The admin repo is not ready"; sleep 1; done &&\
cd $GITDIR_BASE_DIR/admin/admin.git &&\
until git log >> /dev/null 2>&1; do echo "The admin repo is been initalized, waiting"; sleep 5; done; \
mv $GITDIR_BASE_DIR/admin/admin.git/hooks $GITDIR_BASE_DIR/admin/admin.git/hooks.disabled &&\
git clone $GITDIR_BASE_DIR/admin/admin.git $GITDIR_BASE_DIR/admin/unbared &&\
cd $GITDIR_BASE_DIR/admin/unbared &&\
git config user.name "gitdir Docker Helper Script" &&\
git config user.email "gitdir@gitdir.nosuchtld" &&\
vi config.yml &&\
git commit --no-verify config.yml &&\
git push &&\
rm -rf $GITDIR_BASE_DIR/admin/unbared &&\
echo "Saved! You may need to restart the server to verify and apply the settings"; \
mv $GITDIR_BASE_DIR/admin/admin.git/hooks.disabled $GITDIR_BASE_DIR/admin/admin.git/hooks \
)' > /usr/bin/gitdir_config && chmod +x /usr/bin/gitdir_config
WORKDIR $GITDIR_BASE_DIR
USER gitdir:gitdir

EXPOSE 2222
ENTRYPOINT ["gitdir"]
CMD []
