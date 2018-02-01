[![Build Status](https://travis-ci.org/nmaupu/dell-provisioner.svg?branch=master)](https://travis-ci.org/nmaupu/dell-provisioner)
[![Go Report Card](https://goreportcard.com/badge/github.com/nmaupu/dell-provisioner)](https://goreportcard.com/report/github.com/nmaupu/dell-provisioner)

# What is dell-provisioner ?

Dell-provisioner is a Kubernetes external provisioner. It creates / deletes volume and associates LUN using smcli command line tool on the flow on a remote SAN whenever a `PersistentVolumeClaim` appears on the cluster.

# Known compatibility

- Dell MD3200i

# Building

```
make vendor && make
```

Binary is located into `bin/dell-provisioner-linux-amd64`
Building for mac OS is also possible

# Usage

## Provision the provisioner

You need an image with *Dell smcli* available to be able to use dell-provisioner.
Dockerfile example :
```
FROM centos AS builder
RUN  yum -y install epel-release && \
     yum -y install xorriso && \
     curl -o /dell.iso https://downloads.dell.com/FOLDER04066625M/1/DELL_MDSS_Consolidated_RDVD_6_5_0_1.iso && \
     osirrox -indev /dell.iso -extract /linux/mdsm/SMIA-LINUXX64.bin /dell.bin && \
     rm -f /dell.iso

FROM centos AS installed
COPY --from=builder /dell.bin /
COPY installer.properties /
RUN  yum -y install unzip && \
     /dell.bin -f /installer.properties

FROM centos
RUN  mkdir -p /opt/smcli/mdstoragemanager/client && \
     mkdir -p /var/opt/SM && \
     echo "BASEDIR=/opt/smcli/mdstoragemanager" > /var/opt/SM/LAUNCHER_ENV
COPY --from=installed /opt/dell/mdstoragemanager/jre /opt/smcli/mdstoragemanager/jre
COPY --from=installed /opt/dell/mdstoragemanager/client/SMcli /opt/smcli/mdstoragemanager/client/
COPY --from=installed /opt/dell/mdstoragemanager/client/*.jar /opt/smcli/mdstoragemanager/client/

ENTRYPOINT ["/opt/smcli/mdstoragemanager/client/SMcli"]
```

Once this is available, use the following pod to deploy dell-provisioner :
```
kind: Pod
apiVersion: v1
metadata:
  name: dell-provisioner
  namespace: kube-system
spec:
  containers:
    - name: dell-provisioner
      image: <my-image>
      imagePullPolicy: Always
      env:
        - name: IDENTIFIER
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        - name: SAN_ADDRESS
          value: 192.168.10.10
        - name: SAN_PASSWORD
          value: mypass
        - name: SAN_GROUP_NAME
          value: k8s
        - name: SMCLI_COMMAND
          value: /path/to/smcli
```

Use a better secret handling for a real usage.

If the cluster uses RBAC, you need to create according roles and bindings :
```
kubectl apply -f rbac.yaml
```

# Example usage

We first need a *storage class* which tells which provisioner to use and how:

```
---
kind: StorageClass
apiVersion: storage.k8s.io/v1beta1
metadata:
  name: dell-provisioner-sc
provisioner: maupu.org/dell-provisioner
parameters:
  targetPortal: 192.168.99.100:3260
  portals: 192.168.99.101:3260,192.168.99.102:3260
  iqn: iqn.2003-01.org.linux-iscsi.minishift:targetd
  volumeGroup: k8s
  initiators: iqn.2017-04.com.example:node1
  fsType: ext4
  readonly: false
```

See this for a more complete example: https://github.com/kubernetes-incubator/external-storage/blob/master/iscsi/targetd/kubernetes/iscsi-provisioner-class.yaml

Next, create a persistent volume claim using that storage class :

```
---
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: dell-provisioner-test-pvc
spec:
  storageClassName: dell-provisioner-test-sc
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
```

Use that claim on a testing pod :

```
---
kind: Pod
apiVersion: v1
metadata:
  name: dell-provisioner-test-pod
spec:
  containers:
  - name: dell-provisioner-test-pod
    image: gcr.io/google_containers/busybox:1.24
    command:
      - "/bin/sh"
    args:
      - "-c"
      - "date >> /mnt/file.log && exit 0 || exit 1"
    volumeMounts:
      - name: dell-provisioner-test-volume
        mountPath: "/mnt"
  restartPolicy: "Never"
  volumes:
    - name: dell-provisioner-test-volume
      persistentVolumeClaim:
        claimName: dell-provisioner-test-pvc
```

The underlying volume should be create and associated with your pod. Remark: Kubernetes iSCSI implementation will format your volumes when firtly used.
Beware though, when the persistent volume claim is deleted, so is your volume !

Logs can be accessed using:

```
kubectl -n kube-system logs -f dell-provisioner
```
