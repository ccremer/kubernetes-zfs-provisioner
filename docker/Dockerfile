FROM docker.io/library/alpine:3.16 as runtime

ENTRYPOINT ["kubernetes-zfs-provisioner"]

RUN \
    apk add --no-cache curl bash openssh && \
    adduser -S zfs -G root

COPY docker/zfs.sh /usr/bin/zfs
COPY docker/update-permissions.sh /usr/bin/update-permissions
COPY kubernetes-zfs-provisioner /usr/bin/

USER zfs:root
