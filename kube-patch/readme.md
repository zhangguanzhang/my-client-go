
利用cronjob自动补全二进制或者kubeadm集群下管理组件+kubelet+kubep-roxy的svc和ep信息
```
~ # kube-patch --help
Usage of kube-patch:
      --apiserver-host string        The address of the Kubernetes Apiserver to connect to in the format of protocol://address:port, e.g., http://localhost:8080. If not specified, the assumption is that the binary runs inside a Kubernetes cluster and local discovery is attempted.
      --controller-port int32        the metrics port of the kube-controller-manager (default 10252)
      --controller-svc-name string   the svc name of the kube-controller-manager (default "kube-controller-manager")
      --kubeconfig string            absolute path to the kubeconfig file
      --kubelet                      enable patch the kubelet (default true)
      --kubelet-port int32           the metrics port of the kubelet (default 10250)
      --kubelet-svc-name string      the svc name of the kubelet (default "kubelet")
      --kubeproxy                    enable patch the kube-proxy (default true)
      --kubeproxy-port int32         the metrics port of the kube-proxy (default 10249)
      --kubeproxy-svc-name string    the svc name of the kube-proxy (default "kube-proxy")
      --scheduler-port int32         the metrics port of the kube-scheduler (default 10251)
      --scheduler-svc-name string    the svc name of the kube-scheduler (default "kube-scheduler")
      --timeout int                  connect timeout (default 5)
pflag: help requested
```
