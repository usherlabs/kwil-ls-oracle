
```shell
kwil-cli database drop demo --sync
kwil-cli database deploy --sync -p=./demo.kf --name=demo --sync
```

create a new database that won't receive it because it's not included in config
```shell
kwil-cli database drop demo_not_included --sync
kwil-cli database deploy --sync -p=./demo.kf --name=demo_not_included --sync
```

```shell
kwil-cli database call -a=test -n=demo
```

```shell
kwil-cli database call -a=get_data -n=demo
```

```shell
kwil-cli database call -a=get_data -n=demo_not_included
```
