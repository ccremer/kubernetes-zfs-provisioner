# kubernetes-zfs-provisioner

zfs-provisioner is an out of cluster external provisioner for Kubernetes. It creates ZFS datasets and shares them via NFS to make them mountable to pods. Currently all ZFS attributes are inherited from the parent dataset, different storage classes for e.g. cached/non-cached datasets or manually setting attributes via annotations should follow in the future. This provisioner is considered highly **experimental** and is still under development.

For more information about external storage in kubernetes, see [kubernetes-incubator/external-storage](https://github.com/kubernetes-incubator/external-storage).

## Usage
The provisioner can be configured via the following environment variables:

| Variable | Description | Default |
| :------: | :---------- | :-----: |
| `ZFS_PARENT_DATASET` | The parent dataset in which datasets will be created, needs to exist beforehand. No leading or trailing slashes. Mandatory. | |
| `ZFS_SHARE_OPTIONS` | Additional nfs share options, comma-separated. | `rw=@10.0.0.0/8` |
| `ZFS_SERVER_HOSTNAME` | The hostname or ip which the pods should use to mount the volume. Determined via `hostname -f` if empty. | |
| `ZFS_PROVISIONER_NAME` | Name of the provisioner. Change only if you want to run multiple instances. | `gentics.com/zfs` |
| `ZFS_KUBE_RECLAIM_POLICY` | The reclaim policy to use, currently either `Delete` or `Retain`. |`Delete` |
| `ZFS_KUBE_CONF` | Path to the kubernetes config file which will be used to connect to the cluster. |`kube.conf` |
| `ZFS_METRICS_PORT` | Port on which to export Prometheus metrics. | `8080` |

## Notes
### Reclaim policy
This provisioner currently supports the `Delete` or `Retain` reclaim policy. Until [kubernetes/#38192](https://github.com/kubernetes/kubernetes/issues/38192) is resolved, this is configured per provisioner via an environment variable. To use both, run two instances of the provisioner and configure different storage classes.

### Storage space
The provisioner uses the `reflimit` and `refquota` ZFS attributes to limit storage space for volumes. Each volume can not use more storage space than the given resource request and also reserves exactly that much. This means that over provisioning is not possible. Snapshots **do not** account for the storage space limit. See Oracles [ZFS Administration Guide](https://docs.oracle.com/cd/E23823_01/html/819-5461/gazvb.html) for more information.

## Development

The tests need to manage ZFS datasets, create a testing pool on a disk image:

```
# Create a 10GB disk image
dd if=/dev/zero bs=1024m count=10 of=disk.img
```

### Linux

```
runcate --size 1G disk1.img
sudo zpool create pool1 $PWD/disk1.img -m $PWD/test
```

### Mac

```
# Mount the image as a block device, MacOS way
hdiutil attach -imagekey diskimage-class=CRawDiskImage -nomount disk.img
# Create zpool with mount in current directory
sudo zpool create -m $PWD/test -f test /dev/disk2
```
For development under other operating systems, adapt mount command and block device.

## Building

You need GO and go-dep (the example below works on apt based systems)

```
sudo apt install golang-go go-dep
```

In order to checkout all of the dependencies you also need git and mercurial (again examples are for installing from an apt based system)

```
sudo apt install git mercurial
```

Now you need to use git to clone the repo onto your local filesystem lets call that location PATH_TO_REPO. Once you have cloned the repo, you need to setup the GO path.

```
# If $GOPATH is empty
mkdir -p ~/go
export GOPATH=$HOME/go

mkdir -p $GOPATH/src
ln -s $PATH_TO_REPO $GOPATH/src/kubernetes-zfs-provisioner
cd $GOPATH/src/kubernetes-zfs-provisioner

# Install dependencies
dep ensure

# Build
make build
```

## Deployment / Installing
The result of the build process is a binary file in the 'bin' directory named 'zfs-provisioner'.
This binary needs to be running (in some fashion) on each host node you are going to use. There are two methods for doing
this an "internal to cluster" method and an "external to cluster" method. Regardless of which method is used, you must
have an underlying host that is capable of using zfs (which includes the kernel modules and user space programs).
Both methods have common ending steps of creating a storage class and configuring pods to use persistent volume claims.


### External to Cluster
This method interacts directly with the hosts and is the least automated because it requires the most administrator work,
however it is currently the simplest. This method is accomplished by installing the binary directly to each host and
setting up a service to start the binary on host boot.

We assume that you are using a systemd based host distro (like [CoreOS](https://coreos.com/os/docs/latest/getting-started-with-systemd.html)).
* On the host we are going to use to house our zfs filesystem, create the systemd service file /etc/systemd/system/zfs-provisioner.service with the following content (be sure to change the variables according to your needs):
```
[Unit]
Description=kubernetes zfs provisioner
After=nfs-kernel-server.service
After=kubelet.service

[Service]
TimeoutStartSec=0
Restart=on-failure
Environment=ZFS_PARENT_DATASET=storage/kubernetes/pv
Environment=ZFS_SHARE_SUBNET=1.2.3.0/25
Environment=ZFS_SERVER_HOSTNAME=1.2.3.4
Environment=ZFS_KUBE_CONF=/etc/kubernetes/admin.conf
Environment=ZFS_KUBE_RECLAIM_POLICY=Retain
Environment=ZFS_METRICS_PORT=8081
ExecStart=/usr/local/bin/zfs-provisioner

[Install]
WantedBy=multi-user.target
```
* Copy bin/zfs-provisioner from your build result to the host system (most likely with an scp command) and ensure it is in /usr/local/bin (or wherever you ExecStart points to)
* Copy your kubernetes configuration file that allows administration of the cluster to /etc/kubernetes (or wherever ZFS_KUBE_CONF points)
* Ensure that your zfs parent dataset exists and is mounted (and set to mount on reboot... which should be the default)
* Enable your zfs-provisioner service (from a shell on the host where zfs-provisioner is now installed do)
```
sudo systemctl enable /etc/systemd/system/zfs-provisioner.service
sudo start zfs-provisioner
```
* Check to ensure it is running correctly
```
sudo systemctl status zfs-provisioner
```

### Internal to Cluster
The idea behind this install method is to create a single instance of a pod on kubernetes where this pod is always executed
on the same host. We do this with a [StatefulSet](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/).
The statefulset ensures that we always get a pod that runs the zfs-provisioner on the host with the parent dataset, this
pod is what then exports the zfs filesystem via NFS such that other pods in the cluster can make persistent volume claims
to the NFS export.

This method involves:
1. create zfs-provisioner.sh this script will
    1. check and see if the desired zfs parent dataset is available and if not try to create it
    2. set the ZFS_PARENT_DATASET, ZFS_SERVER_HOSTNAME, etc. environment variables
    3. if the environment variables are set and the zfs parent dataset exists and is mounted, it will call the
    zfs-provisioner binary, or fail with exit code 1 and echo a message indicating why
2. creating a docker image with
    1. a shell script zfs-provisioner.sh as its entry point
    2. a copy of bin/zfs-provisioner next to zfs-provisioner.sh
    3. exposed NFS related ports
3. pushing that docker image to a registry from which your kubernetes cluster can download it and execute it in a pod
4. installing a single replica statefulset into your kubernetes cluster
This install method is still very experimental (indeed the instructions are no yet complete), so if you are not
comfortable with the outline above, we suggest using the external to cluster method outlined above.

### Creating Storage Class and Configuring Pods
Once you have a running zfs-provisioner it is a straightforward process to use it. We supply some example definitions
in the 'example' directory. In this example we create a [namespace](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/)
for all our zfs related things to live in, followed by a
[storage class](https://kubernetes.io/docs/concepts/storage/storage-classes/) named 'zfs-sc', then we create a
[persistent volume claim](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistentvolumeclaims)
named 'example-pvc' and finally, we create a pod named 'example' that mounts a
[volume](https://kubernetes.io/docs/concepts/storage/volumes/) named 'zfs-volume' which it mounts to /mnt via a volumeMount.

In this install example we are using host XXX.YYY.ZZZ and we have already created a zpool named "kubernetes-mirror". We
use the following environment variables when running zfs-provisioner on XXX.YYY.ZZZ in this example:
```
ZFS_PARENT_DATASET=kubernetes-mirror/pvc
ZFS_PROVISIONER_NAME=gentics.com/zfs-mirror
ZFS_KUBE_CONF=kube_config_my-cluster.yml
```
Your environment will of course be different depending on your use case.

From a system that has kubectl and can be used to make changes to the cluster we do the following (from the base of the repo checkout):
1. Create a namespace (named zfs-system)
    ```
    $ kubectl apply -f zfs-namespace.yml 
    namespace/zfs-system created
    ```
2. Create a storage class that uses our new zfs-provisioner
    ```
    $ kubectl apply -f storageclass.yml 
    storageclass.storage.k8s.io/zfs-sc created
    ```
    This storage class being backed by zfs acts like a "pool", each new persistent volume claim carves out a slice of the pool.
3. Create a persistent volume claim definition that uses the new zfs-provisioner
    ```
    $ kubectl apply -f pvc.yml 
    persistentvolumeclaim/example-pvc created
    ```
    This persistent volume claim is named "example-pvc" and that is the name we use to work with it, and to assign pods
    to it. It is up to the designer of the architecture being deployed on top of kubernetes to correctly assign persistent
    volume claims to pods. Because this PVC is being exported via NFS the same PVC can be shared by many individual pods.
    This means that pods executing concurrently can share data via a single PVC (NFS doesn't always preform well but it will work).
    It also means that a pod can be moved from one host to another and still have access to the same data.
4. Check to see that our new persistent volume claim worked as expected (in the below example below it did not)
    ```
    $ kubectl --namespace=zfs-system describe pvc example-pvc
    Name:          example-pvc
    Namespace:     zfs-system
    StorageClass:  zfs-sc
    Status:        Pending
    Volume:        
    Labels:        <none>
    Annotations:   control-plane.alpha.kubernetes.io/leader:
                     {"holderIdentity":"e341e188-1f7e-11e9-a563-e0db5570db25","leaseDurationSeconds":15,"acquireTime":"2019-01-24T02:34:30Z","renewTime":"2019-...
                   kubectl.kubernetes.io/last-applied-configuration:
                     {"apiVersion":"v1","kind":"PersistentVolumeClaim","metadata":{"annotations":{"volume.beta.kubernetes.io/storage-class":"zfs-sc"},"name":"e...
                   volume.beta.kubernetes.io/storage-class: zfs-sc
                   volume.beta.kubernetes.io/storage-provisioner: gentics.com/zfs-mirror
    Finalizers:    [kubernetes.io/pvc-protection]
    Capacity:      
    Access Modes:  
    Events:
      Type     Reason              Age    From                                                                             Message
      ----     ------              ----   ----                                                                             -------
      Warning  ProvisioningFailed  4m32s  gentics.com/zfs-mirror XXX.YYY.ZZZ e341e188-1f7e-11e9-a563-e0db5570db25  Failed to provision volume with StorageClass "zfs-sc": Creating ZFS dataset failed with: exit status 1: "/sbin/zfs zfs create -o sharenfs=rw=@10.0.0.0/8 -o refquota=1000000 -o refreservation=1000000 kubernetes-mirror/pvc/pvc-9261ab2a-1f80-11e9-bf82-e0db5570db25" => cannot share 'kubernetes-mirror/pvc/pvc-9261ab2a-1f80-11e9-bf82-e0db5570db25': share(1M) failed
    filesystem successfully created, but not shared
      Normal     Provisioning          4m24s (x2 over 4m34s)  gentics.com/zfs-mirror XXX.YYY.ZZZ e341e188-1f7e-11e9-a563-e0db5570db25  External provisioner is provisioning volume for claim "zfs-system/example-pvc"
      Warning    ProvisioningFailed    4m24s                  gentics.com/zfs-mirror XXX.YYY.ZZZ e341e188-1f7e-11e9-a563-e0db5570db25  Failed to provision volume with StorageClass "zfs-sc": Creating ZFS dataset failed with: exit status 1: "/sbin/zfs zfs create -o sharenfs=rw=@10.0.0.0/8 -o refquota=1000000 -o refreservation=1000000 kubernetes-mirror/pvc/pvc-9261ab2a-1f80-11e9-bf82-e0db5570db25" => cannot create 'kubernetes-mirror/pvc/pvc-9261ab2a-1f80-11e9-bf82-e0db5570db25': dataset already exists
      Normal     ExternalProvisioning  13s (x22 over 4m36s)   persistentvolume-controller                                                      waiting for a volume to be created, either by external provisioner "gentics.com/zfs-mirror" or manually created by system administrator
    Mounted By:  <none>
    ```
    In the above example, the provisioning failed due to a failure of the host system XXX.YYY.ZZZ to create the NFS share,
    but it did create the zfs filesystem. Kubernetes retired creating the pvc and it failed the second time because the
    ZFS dataset already exists.
    
    A correct execution looks like the following:
    ```
    $ kubectl --namespace=zfs-system describe pvc example-pvc
    Name:          example-pvc
    Namespace:     zfs-system
    StorageClass:  zfs-sc
    Status:        Bound
    Volume:        pvc-95e1b736-2034-11e9-9e68-e0db5570db25
    Labels:        <none>
    Annotations:   control-plane.alpha.kubernetes.io/leader:
                     {"holderIdentity":"8b8d670c-2034-11e9-b753-e0db5570db25","leaseDurationSeconds":15,"acquireTime":"2019-01-25T00:03:06Z","renewTime":"2019-...
                   kubectl.kubernetes.io/last-applied-configuration:
                     {"apiVersion":"v1","kind":"PersistentVolumeClaim","metadata":{"annotations":{"volume.beta.kubernetes.io/storage-class":"zfs-sc"},"name":"e...
                   pv.kubernetes.io/bind-completed: yes
                   pv.kubernetes.io/bound-by-controller: yes
                   volume.beta.kubernetes.io/storage-class: zfs-sc
                   volume.beta.kubernetes.io/storage-provisioner: gentics.com/zfs-mirror
    Finalizers:    [kubernetes.io/pvc-protection]
    Capacity:      1M
    Access Modes:  RWX
    Events:
      Type       Reason                 Age                From                                                                             Message
      ----       ------                 ----               ----                                                                             -------
      Normal     Provisioning           36s                gentics.com/zfs-mirror XXX.YYY.ZZZ 8b8d670c-2034-11e9-b753-e0db5570db25  External provisioner is provisioning volume for claim "zfs-system/example-pvc"
      Normal     ExternalProvisioning   35s (x4 over 39s)  persistentvolume-controller                                                      waiting for a volume to be created, either by external provisioner "gentics.com/zfs-mirror" or manually created by system administrator
      Normal     ProvisioningSucceeded  35s                gentics.com/zfs-mirror XXX.YYY.ZZZ 8b8d670c-2034-11e9-b753-e0db5570db25  Successfully provisioned volume pvc-95e1b736-2034-11e9-9e68-e0db5570db25
    Mounted By:  <none>
    ```
5. Once we have a working pvc, we create a pod that uses our new persistent volume claim
    ```
    kubectl apply -f pod.yml
    ```
You should be able to check the status of the pvc using the "describe" feature and check the status of the pod via:
```
$ kubectl --namespace=zfs-system describe pod example
Name:               example
Namespace:          zfs-system
...
...
Events:
  Type    Reason     Age    From                    Message
  ----    ------     ----   ----                    -------
  Normal  Scheduled  3m33s  default-scheduler       Successfully assigned zfs-system/example to 192.168.20.22
  Normal  Pulling    3m25s  kubelet, 192.168.20.22  pulling image "busybox"
  Normal  Pulled     3m18s  kubelet, 192.168.20.22  Successfully pulled image "busybox"
  Normal  Created    3m12s  kubelet, 192.168.20.22  Created container
  Normal  Started    3m12s  kubelet, 192.168.20.22  Started container
```
It may take some time for the pod to completely start up as it may have to pull a busybox image.

The first time this example pod is executed it should output a log indicating that the file it is supposed to use does not
exist and that it is going to create it.
```
$ kubectl --namespace=zfs-system logs example
file does not exist attempting to create it
root
```
The exit status of that pod will indicate if the file was created or not.
```
$ kubectl --namespace=zfs-system describe pod example
Name:               example
Namespace:          zfs-system
...
...
State:          Terminated
  Reason:       Completed
  Exit Code:    0
```
Exit code 0 means that the file /mnt/SUCCESS was created. Now delete the pod and recreate it. Because we still have the pvc
we expect this second pod to already have the file /mnt/SUCCESS.
```
$ kubectl --namespace=zfs-system delete pod example
pod "example" deleted
$ kubectl --namespace=zfs-system get pods
No resources found.
$ kubectl apply -f pod.yml 
pod/example created
$ kubectl --namespace=zfs-system get pods
NAME      READY   STATUS              RESTARTS   AGE
example   0/1     ContainerCreating   0          7s
```
Wait for the pod to correctly deploy and complete executing and then check its logs.
```
$ kubectl --namespace=zfs-system logs example
file already exists
```
At this point everything is shown to be working, and you need to adapt the example pod to your application needs.