# GitOps-Friendly MachineSets Operator

MachineSets created by the OpenShift installer include a random string of characters in their names. For example, after you deploy an OpenShift cluster on AWS, you may find three MachineSets named like this:

```
$ oc get machineset -n openshift-machine-api
NAME                                   DESIRED   CURRENT   READY   AVAILABLE   AGE
cluster-3af1-6cnhg-worker-us-east-2a   0         0                             2d5h
cluster-3af1-6cnhg-worker-us-east-2b   0         0                             2d5h
cluster-3af1-6cnhg-worker-us-east-2c   0         0                             2d5h
```

The three MachineSets start with `cluster-3af1-6cnhg` which is called an _infrastructure name_ and unfortunately is generated randomly by the OpenShift installer. These MachineSets are difficult to manage using GitOps as their names cannot be determined ahead of time. In addition to this, the same infrastructure name is also used in the defition of these MachineSets, for example:

<pre>
$ oc get machineset -n openshift-machine-api cluster-3af1-6cnhg-worker-us-east-2a -o yaml
apiVersion: machine.openshift.io/v1beta1
kind: MachineSet
metadata:
  annotations:
    autoscaling.openshift.io/machineautoscaler: openshift-machine-api/<b>cluster-3af1-6cnhg</b>-worker-us-east-2a
  creationTimestamp: "2021-11-20T18:22:10Z"
  generation: 8
  labels:
    machine.openshift.io/cluster-api-cluster: <b>cluster-3af1-6cnhg</b>
  name: <b>cluster-3af1-6cnhg</b>-worker-us-east-2a
  namespace: openshift-machine-api
  
  ...
</pre>

## How Does the GitOps-Friendly MachineSets Operator Help?

The GitOps-Friendly MachineSets Operator helps in two steps:

1. The operator allows you to create MachineSets without the need to supply the cluster-specific infrastructure name. Instead, you insert a special token `INFRANAME` into your MachineSet definition, which will be replaced with the real infrastructure name by the operator.

2. As soon as the first node created by your MachineSet becomes available, the operator will scale the installer-provisioned MachineSets to zero. They cannot be managed by GitOps anyway.

The operator is tested on AWS and vSphere OpenShift clusters, however, it should work with any underlying infrastructure provider.

## Building Container Images (Optional)

This section provides instructions for building your custom operator images. If you'd like to deploy the operator using pre-built images, you can continue to the next section.

Set the version you want to build. See `git tag` for available versions:

```
$ VERSION=0.1.0
```

Check out the tag:

```
$ git checkout $VERSION
```

Set the custom image names. Replace the image names below with your own:

```
$ IMG=quay.io/noseka1/gitops-friendly-machinesets-operator:$VERSION
$ IMAGE_TAG_BASE=quay.io/noseka1/gitops-friendly-machinesets-operator
```

Build operator image:

```
$ make docker-build IMG=$IMG
```

Push the finished operator image to the image registry:

```
$ podman push $IMG
```

Generate operator bundle artifacts:

```
$ make bundle IMG=$IMG CHANNELS=stable DEFAULT_CHANNEL=stable
```

Build bundle container image:

```
$ make bundle-build IMAGE_TAG_BASE=$IMAGE_TAG_BASE
```

Push bundle image to registry:

```
$ podman push $IMAGE_TAG_BASE-bundle:v$VERSION
```

Build catalog container image:

```
$ make catalog-build IMAGE_TAG_BASE=$IMAGE_TAG_BASE
```

Push catalog image to registry:

```
$ podman push $IMAGE_TAG_BASE-catalog:v$VERSION
```

## Deploying the Operator

If you'd like to deploy the operator using your custom built images, substitute the name of your catalog image in `deploy/gitops-friendly-machinesets-catsrc.yaml`:

```
$ sed -i "s#image:.*#image: $IMAGE_TAG_BASE-catalog:v$VERSION#" deploy/gitops-friendly-machinesets-catsrc.yaml
```

Alternatively, you can leverage the pre-built operator images to deploy the operator:

```
$ sed -i "s#image:.*#image: quay.io/noseka1/gitops-friendly-machinesets-operator-catalog:v0.1.0#" deploy/gitops-friendly-machinesets-catsrc.yaml
```

Deploy the operator:

```
$ oc apply -k deploy
```

## Creating MachineSets

Create a MachineSet specific to your underlying infrastructure provider. For example, a MachineSet for AWS and vSphere may look like the ones below. Note that all occurences of the infrastructure name are marked using the `INFRANAME` token. Operator will replace this `INFRANAME` token with the real infrastructure name after the MachineSet manifest is applied.

Also note that you must add two _annotations_ that are required for the operator to take any action on the MachineSet:
1. Set `metadata.annotations.gitops-friendly-machinesets.redhat-cop.io/enabled: "true"`
2. Set `spec.template.metadata.annotations.gitops-friendly-machinesets.redhat-cop.io/enabled: "true"`

### Sample AWS MachineSet

<pre>
apiVersion: machine.openshift.io/v1beta1
kind: MachineSet
metadata:
  <b>annotations:
    gitops-friendly-machinesets.redhat-cop.io/enabled: "true"</b>    
  labels:
    machine.openshift.io/cluster-api-cluster: <b>INFRANAME</b>
  name: mymachineset
  namespace: openshift-machine-api
spec:
  replicas: 3
  selector:
    matchLabels:
      machine.openshift.io/cluster-api-cluster: <b>INFRANAME</b>
      machine.openshift.io/cluster-api-machineset: mymachineset
  template:
    metadata:
      <b>annotations:
        gitops-friendly-machinesets.redhat-cop.io/enabled: "true"</b>
      labels:
        machine.openshift.io/cluster-api-cluster: <b>INFRANAME</b>
        machine.openshift.io/cluster-api-machine-role: worker
        machine.openshift.io/cluster-api-machine-type: worker
        machine.openshift.io/cluster-api-machineset: mymachineset
    spec:
      metadata: {}
      providerSpec:
        value:
          ami:
            id: ami-03d9208319c96db0c
          apiVersion: awsproviderconfig.openshift.io/v1beta1
          blockDevices:
          - ebs:
              encrypted: true
              iops: 0
              kmsKey:
                arn: ""
              volumeSize: 120
              volumeType: gp2
          credentialsSecret:
            name: aws-cloud-credentials
          deviceIndex: 0
          iamInstanceProfile:
            id: <b>INFRANAME</b>-worker-profile
          instanceType: m5.xlarge
          kind: AWSMachineProviderConfig
          metadata:
            creationTimestamp: null
          placement:
            availabilityZone: us-east-2a
            region: us-east-2
          securityGroups:
          - filters:
            - name: tag:Name
              values:
              - <b>INFRANAME</b>-worker-sg
          subnet:
            filters:
            - name: tag:Name
              values:
              - <b>INFRANAME</b>-private-us-east-2a
          tags:
          - name: kubernetes.io/cluster/<b>INFRANAME</b>
            value: owned
          userDataSecret:
            name: worker-user-data
</pre>

### Sample vSphere MachineSet

<pre>
apiVersion: machine.openshift.io/v1beta1
kind: MachineSet
metadata:
  <b>annotations:
    gitops-friendly-machinesets.redhat-cop.io/enabled: "true"</b>
  labels:
    machine.openshift.io/cluster-api-cluster: <b>INFRANAME</b>
  name: mymachineset
  namespace: openshift-machine-api
spec:
  replicas: 3
  selector:
    matchLabels:
      machine.openshift.io/cluster-api-cluster: <b>INFRANAME</b>
      machine.openshift.io/cluster-api-machineset: mymachineset
  template:
    metadata:
      <b>annotations:
        gitops-friendly-machinesets.redhat-cop.io/enabled: "true"</b>
      labels:
        machine.openshift.io/cluster-api-cluster: <b>INFRANAME</b>
        machine.openshift.io/cluster-api-machine-role: worker
        machine.openshift.io/cluster-api-machine-type: worker
        machine.openshift.io/cluster-api-machineset: mymachineset
    spec:
      metadata: {}
      providerSpec:
        value:
          apiVersion: vsphereprovider.openshift.io/v1beta1
          credentialsSecret:
            name: vsphere-cloud-credentials
          diskGiB: 120
          kind: VSphereMachineProviderSpec
          memoryMiB: 32768
          metadata:
            creationTimestamp: null
          network:
            devices:
            - networkName: OpenShift Network
          numCPUs: 4
          numCoresPerSocket: 2
          snapshot: ""
          template: <b>INFRANAME</b>-rhcos
          userDataSecret:
            name: worker-user-data
          workspace:
            datacenter: Datacenter
            datastore: datastore1
            folder: /Datacenter/vm/mycluster
            resourcePool: /Datacenter/host/Cluster/Resources
            server: photon-machine.lab.example.com
</pre>
