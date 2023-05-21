## Secret Syncer VCluster Plugin

This plugin is used to sync desired `secrets` from `host cluster` to `vcluster`. The `secrets` are synced from the namespace whene `vcluster` is installed into.

## How to Install Plugin

To use the `plugin`, create or update `vluster` using `plugin.yaml` file. The `plugin.yaml` file is used to confgiure `vlcuster` to use `Secret Syncer plugin`.

```bash
vcluster create my-vcluster -n ssyncer -f https://raw.githubusercontent.com/araminian/vcluster-secret-syncer/main/plugin.yaml
```

This command will create a `vcluster` named `my-vcluster` in namespace `ssyncer` and will use `Secret Syncer plugin` to sync  desired `secrets` from `host cluster` to `vcluster`.

## How to Use Plugin

To sync **any secrets** from `vcluster host namespace` to **any namespace** in `vcluster`, we are using following `annotations` which both are **required**.

```yaml
    "secret-syncer.cloudarmin.me/enabled": "true"
    "secret-syncer.cloudarmin.me/destination-namespace": ""
```

The `secret-syncer.cloudarmin.me/enabled` annotation is used to let `Secret Syncer plugin` know that this `secret` should be synced to `vcluster`.

The `secret-syncer.cloudarmin.me/destination-namespace` annotation is used to specify the `namespace` in `vcluster` that the `secret` should be synced to.

### Example: Syncing Secret to `default` Namespace in `vcluster`

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: mysecret
  namespace: ssyncer # vcluster host namespace
  annotations:
    # Enable sync for this secret
    "secret-syncer.cloudarmin.me/enabled": "true"
    # Secret will be synced to `default` namespace in `vcluster`
    "secret-syncer.cloudarmin.me/destination-namespace": "default"
type: Opaque
data:
  username: YWRtaW4=
  password: YXJtaW4=
```

### Example: Syncing Secret to `my-namespace` Namespace in `vcluster` Which Not Exists

In this example `my-namespace` namespace does not exists in `vcluster`. In this scenario `Secret Syncer` can't sync the `secret` since the `destination namesapce` does not exists.

This secret marked as `failed` and `Secret Syncer` will retry to sync this `secret` every `30s`. Once the `my-namespace` namespace created in `vcluster`, `Secret Syncer` will sync the `secret` to `my-namespace` namespace.


```yaml
apiVersion: v1
kind: Secret
metadata:
  name: mysecret
  namespace: ssyncer # vcluster host namespace
  annotations:
    # Enable sync for this secret
    "secret-syncer.cloudarmin.me/enabled": "true"
    # Secret will be synced to `my-namesapce` namespace in `vcluster`
    "secret-syncer.cloudarmin.me/destination-namespace": "my-namesapce"
type: Opaque
data:
  username: YWRtaW4=
  password: MWYyZDFlMmU2N2Rm
```
