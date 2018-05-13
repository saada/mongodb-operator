# mongodb-operator

Inspired by [Sundeep Dinesh's talk](https://www.youtube.com/watch?v=m8anzXcP-J8) on his experiment called [mongo-k8s-sidecar](https://github.com/thesandlord/mongo-k8s-sidecar).
In this video, he mentions next steps for the project including creating an Operator. This is the manifestation of his vision.

## Stability

Limited Alpha

## Features

* Built with [operator-sdk](https://github.com/operator-framework/operator-sdk)

## Supported Topologies

* Single mongod

```yml
apiVersion: "saada.mongodb.operator/v1alpha1"
kind: "MongoService"
metadata:
  name: "MyMongoService"
spec:
  replicas: 1
```

## Upcoming Features

* Configurable Volumes
* Client driven index creation
* Failover Replicasets
* [Backups](https://medium.com/google-cloud/mgob-a-mongodb-backup-agent-for-kubernetes-cfc9b30c6c92)
* Automatic arbiter pod deployment while a statefulset is down

## Upcoming Deployment Topologies

* Replicaset

```yml
apiVersion: "saada.mongodb.operator/v1alpha1"
kind: "MongoService"
metadata:
  name: "MyMongoReplicasetCluster"
spec:
  replicas: 3
```

* Sharded replicaset

```yml
apiVersion: "saada.mongodb.operator/v1alpha1"
kind: "MongoService"
metadata:
  name: "MyMongoReplicasetCluster"
spec:
  shards: 3
  replicasPerShard: 3
```

* Config Servers

```yml
apiVersion: "saada.mongodb.operator/v1alpha1"
kind: "MongoService"
metadata:
  name: "MyMongoReplicasetCluster"
spec:
  shards: 3
  replicasPerShard: 3
  configServers: 3
```

* Arbiters

Create arbiters when instances fail to come up and even out quorum count to satisfy an odd number of instances. This is to prevent [split-brain](<https://en.wikipedia.org/wiki/Split-brain_(computing)>)

```yml
apiVersion: "saada.mongodb.operator/v1alpha1"
kind: "MongoService"
metadata:
  name: "MyMongoReplicasetCluster"
spec:
  shards: 3
  replicasPerShard: 4
  configServers: 3
  arbiters: true # each of the shards will get 1 arbiter instance to satisfy an odd number of cluster members (4+1)
```

## Development

After installing the [operator-sdk](https://github.com/operator-framework/operator-sdk)

```sh
operator-sdk new mongodb-operator --api-version=saada.mongodb.operator/v1alpha1 --kind=MongoService
```

After updating `types.go`, run

```sh
make regenerate
```

Run makefile and check the operator pod's logs to debug issues

```sh
make
```
