FROM docker.io/library/alpine:3.24

ENTRYPOINT ["kubernetes-zfs-provisioner"]

RUN adduser -S zfs -G root

COPY kubernetes-zfs-provisioner /usr/bin/

USER zfs:root
