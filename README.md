# net-loadtest-go

Deploy manifests in Kubernetes
```
kubectl apply -f ./manifests
```

---

Build the client:
```
make build 
```

Then run it against your ingress address
```
./loadtest client -a "<some-address>"
```   