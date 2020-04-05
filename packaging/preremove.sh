#!/bin/sh

service="zfs-provisioner.service"

/bin/systemctl stop ${service}
/bin/systemctl disable ${service}
