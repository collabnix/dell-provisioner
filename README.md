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
