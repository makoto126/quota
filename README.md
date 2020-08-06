# quota

## Design
![acJMAH.png](https://s1.ax1x.com/2020/08/06/acJMAH.png)
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
Test metrics:
```
curl [pod ip]:8080/metrics
```

## Configuration
Configurable items:
```go
type Config struct {
	NodeName         string        `required:"true" split_words:"true"`
	BaseDir          string        `default:"/data" split_words:"true"`
	AvailableNum     int           `default:"1" split_words:"true"`
	DefaultResync    time.Duration `default:"30s" split_words:"true"`
	ListDuration     time.Duration `default:"5s" split_words:"true"`
    StorageClassName string        `default:"local-storage" split_words:"true"`
	StorageCapacity  string        `split_words:"true"`
	RecordDuration   time.Duration `default:"30s" split_words:"true"`
}
```
Using environment variables, for example:
- AVAILABLE_NUM = 2
- LIST_DURATION = 15s


