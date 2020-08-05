# quota

## Deploy
Execute on each k8s node:
```
rm -f /etc/projects /etc/projid
mount -o prjquota [your data disk] /data
```
Execute on k8s master:
```
kubectl apply -f deploy/deploy.yaml
```

## Test
Test quota:
```
kubectl apply -f test/test.yaml
```
Test resize:
```
kubectl patch pvc test-local-pvc --patch \
'{"metadata": {"annotations": {"quota": "500Mi"}}'
```
