[![Build Status](https://travis-ci.org/nmaupu/dell-provisioner.svg?branch=master)](https://travis-ci.org/nmaupu/dell-provisioner)
[![Go Report Card](https://goreportcard.com/badge/github.com/nmaupu/dell-provisioner)](https://goreportcard.com/report/github.com/nmaupu/dell-provisioner)

# Kubernetes External Storage Provisioner for Dell PowerVault MD3200i

Dell-provisioner is a Kubernetes external provisioner. It creates / deletes volume and associates LUN using smcli command line tool on the flow on a remote SAN whenever a `PersistentVolumeClaim` appears on the cluster.

# Known compatibility

- Dell MD3200i

# Pre-requisite

- Ubuntu 18.04
- Install make
- Install glide

```
sudo add-apt-repository ppa:masterminds/glide && sudo apt-get update
sudo apt-get install glide
```

- Install go 

Run the below command being on home directory

```
sudo wget https://dl.google.com/go/go1.13.1.linux-amd64.tar.gz
sudo tar xvf go1.13.1.linux-amd64.tar.gz
sudo mv go /home/cse/
```


# Building

If you want to build dell-provisioner of your own, follow the below step:

```
make vendor && make
```

```
glide install -v --strip-vcs
[WARN]  The --strip-vcs flag is deprecated. This now works by default.
[WARN]  The name listed in the config file (github.com/nmaupu/dell-provisioner) does not match the current location (.)
[WARN]  Lock file may be out of date. Hash check of YAML failed. You may need to run 'update'
[INFO]  Downloading dependencies. Please wait...
[INFO]  --> Fetching github.com/PuerkitoBio/purell
[INFO]  --> Fetching github.com/davecgh/go-spew
[INFO]  --> Fetching github.com/deckarep/golang-set
[INFO]  --> Fetching github.com/gogo/protobuf
[INFO]  --> Fetching github.com/docker/distribution
[INFO]  --> Fetching github.com/emicklei/go-restful
[INFO]  --> Fetching github.com/ghodss/yaml
[INFO]  --> Fetching github.com/go-openapi/jsonpointer
[INFO]  --> Fetching github.com/go-openapi/jsonreference
[INFO]  --> Fetching github.com/go-openapi/spec
[INFO]  --> Fetching github.com/go-openapi/swag
[INFO]  --> Fetching github.com/jawher/mow.cli
[INFO]  --> Fetching github.com/golang/glog
[INFO]  --> Fetching github.com/golang/groupcache
[INFO]  --> Fetching github.com/google/gofuzz
[INFO]  --> Fetching github.com/kubernetes-incubator/external-storage
[INFO]  --> Fetching github.com/juju/ratelimit
[INFO]  --> Fetching github.com/mailru/easyjson
[INFO]  --> Fetching github.com/pborman/uuid
[INFO]  --> Fetching github.com/dghubble/sling
[INFO]  --> Fetching github.com/PuerkitoBio/urlesc
[INFO]  --> Fetching github.com/Sirupsen/logrus
[INFO]  --> Fetching github.com/spf13/pflag
[INFO]  --> Fetching github.com/spf13/viper
[INFO]  --> Fetching github.com/ugorji/go
[INFO]  --> Fetching golang.org/x/net
[INFO]  --> Fetching golang.org/x/text
[INFO]  --> Fetching gopkg.in/inf.v0
[INFO]  --> Fetching gopkg.in/yaml.v2
[INFO]  --> Fetching k8s.io/apimachinery
[INFO]  --> Fetching k8s.io/client-go
[INFO]  --> Fetching k8s.io/kubernetes
```

Binary is located into `bin/dell-provisioner-linux-amd64`
Building for mac OS is also possible.

# Available Dell Provisioner Binary

You can directly download the binaries release from the Release tab

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
