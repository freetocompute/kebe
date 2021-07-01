# Introduction

Kebe intends to be a full replacement for the Snap Store.

# Quickstart

Once you have an environment setup (for instance using https://github.com/freetocompute/kebe-helm)
then you are ready to initialize your store.

Run:

```shell
go run bin/admin/main.go store -s <database ip> -p <database port> -d <database name> 
-x <database password> -u <database user> -m <minio host> -a <minio access key> -k <minio secret key>
initialize
```

You can also use `go run bin/admin/main.go store --help` for more details.

If you've previously initialized your store, you will need to use `destroy` before you can do it again.

Once you've done that you can browse to your assertions http://cluster-address:30900/minio/root. These
will be used in your patched snapd.

# Development

```
task build-push-redeploy
```

## Requirements

* [Taskfile.dev](taskfile.dev)
* Docker
* Kubernetes w/Minio and Postgres [see kebe-helm](https://github.com/freetocompute/kebe-helm)
* Helm