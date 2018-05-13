# mongodb-operator

Inspired by [Sundeep Dinesh's talk](https://www.youtube.com/watch?v=m8anzXcP-J8) on his experiment called [mongo-k8s-sidecar](https://github.com/thesandlord/mongo-k8s-sidecar).
In this video, he mentions next steps for the project including creating an Operator. This is the manifestation of his vision.

## Features

* Built with [operator-sdk](https://github.com/operator-framework/operator-sdk)

## Upcoming Features

* Client driven index creation
* Failover Replicasets
* Backups
* Automatic arbiter pod deployment while a statefulset is down

## Upcoming Deployment Topologies

* Single mongod
* Replcaset
* Sharded replicaset
* Config
* Arbiter

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
