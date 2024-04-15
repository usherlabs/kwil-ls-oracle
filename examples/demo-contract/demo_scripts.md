
```shell
kwil-cli database drop demo --sync
kwil-cli database deploy --sync -p=./demo.kf --name=demo --sync
```

```shell
kwil-cli database call -a=test -n=demo
```

```shell
kwil-cli database call -a=get_data -n=demo
```