package main

import (
	"encoding/json"
	"fmt"
	flag "github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
	"strconv"
)

const (
	tier             = "control-plane"
	httpMetricsName  = "http-metrics"
	httpsMetricsName = "https-metrics"
	// don't edit this ↓
	controllerName = "kube-controller-manager"
	schedulerName  = "kube-scheduler"
	kubeletName    = "kubelet"
	proxyName      = "kube-proxy"
)

var (
	controllerLabelSelector = map[string]string{
		"tier":      tier,
		"component": controllerName,
	}
	schedulerLabelSelector = map[string]string{
		"tier":      tier,
		"component": schedulerName,
	}
	kubeproxyLabelSelector = map[string]string{
		"k8s-app": proxyName,
	}
)

type PATCH struct {
	clientset  *kubernetes.Clientset
	timeout    *int64
	bin        bool
	masterIPs  []string
	nodeIPs    []string
	controller *svcInfo
	scheduler  *svcInfo
	kubelet    *svcInfo
	kubeproxy  *svcInfo
}

type svcInfo struct {
	svcName       string
	labels        map[string]string
	Ports         []corev1.ServicePort
	labelSelector map[string]string
	Annotations   map[string]string
}

func main() {
	apiserverHost := *flag.String("apiserver-host", "", "The address of the Kubernetes Apiserver "+
		"to connect to in the format of protocol://address:port, e.g., "+
		"http://localhost:8080. If not specified, the assumption is that the binary runs inside a "+
		"Kubernetes cluster and local discovery is attempted.")
	kubeConfigPath := *flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	//if home := homedir.HomeDir(); home != "" {
	//	_, err := os.Stat(filepath.Join(home, ".kube", "config"))
	//	if err != nil && os.IsNotExist(err) {
	//		apiserverHost = "http://localhost:8080"
	//	} else {
	//		kubeConfigPath = *flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	//	}
	//
	//}
	if os.Getenv("KUBECONFIG") != "" {
		kubeConfigPath = os.Getenv("KUBECONFIG")
	}
	timeout           := flag.Int64("timeout", 5, "connect timeout")
	controllerSVCName := *flag.String("controller-svc-name", "kube-controller-manager", "the svc name of the kube-controller-manager")
	controllerPort    := *flag.Int32("controller-port", 10252, "the metrics port of the kube-controller-manager")
	schedulerSVCName  := *flag.String("scheduler-svc-name", "kube-scheduler", "the svc name of the kube-scheduler")
	schedulerPort     := *flag.Int32("scheduler-port", 10251, "the metrics port of the kube-scheduler")
	kubeletSVCName    := *flag.String("kubelet-svc-name", "kubelet", "the svc name of the kubelet")
	kubeletPort       := *flag.Int32("kubelet-port", 10250, "the metrics port of the kubelet")

	EnableKubelet     := *flag.Bool("kubelet", true, "enable patch the kubelet")

	kubeproxySVCName  := *flag.String("kubeproxy-svc-name", "kube-proxy", "the svc name of the kube-proxy")
	kubeproxyPort     := *flag.Int32("kubeproxy-port", 10249, "the metrics port of the kube-proxy")

	EnableKubeProxy   := *flag.Bool("kubeproxy", true, "enable patch the kube-proxy")

	flag.Parse()

	controller := &svcInfo{
		svcName: controllerSVCName,
		labels: map[string]string{
			"k8s-app":                       controllerName,
			"kubernetes.io/cluster-service": "true",
			"kubernetes.io/name":            controllerName,
		},
		Ports: []corev1.ServicePort{
			{
				Name: httpMetricsName,
				Port: controllerPort,
			},
		},
		labelSelector: controllerLabelSelector,
		Annotations: map[string]string{
			"prometheus.io/port":   strconv.FormatInt(int64(controllerPort), 10),
			"prometheus.io/scrape": "true",
		},
	}
	scheduler := &svcInfo{
		svcName: schedulerSVCName,
		labels: map[string]string{
			"k8s-app":                       schedulerName,
			"kubernetes.io/cluster-service": "true",
			"kubernetes.io/name":            schedulerName,
		},
		Ports: []corev1.ServicePort{
			{
				Name: httpMetricsName,
				Port: schedulerPort,
			},
		},
		labelSelector: schedulerLabelSelector,
		Annotations: map[string]string{
			"prometheus.io/port":   strconv.FormatInt(int64(schedulerPort), 10),
			"prometheus.io/scrape": "true",
		},
	}
	kubelet := &svcInfo{
		svcName: kubeletSVCName,
		labels: map[string]string{
			"k8s-app":                       kubeletName,
			"kubernetes.io/cluster-service": "true",
			"kubernetes.io/name":            kubeletName,
		},
		Ports: []corev1.ServicePort{
			{
				Name: httpsMetricsName, // https
				Port: kubeletPort,
			},
		},
		Annotations: map[string]string{
			"prometheus.io/port":   strconv.FormatInt(int64(kubeletPort), 10),
			"prometheus.io/scrape": "true",
		},
	}

	kubeProxy := &svcInfo{
		svcName: kubeproxySVCName,
		labels: map[string]string{
			"k8s-app":                       proxyName,
			"kubernetes.io/cluster-service": "true",
			"kubernetes.io/name":            proxyName,
		},
		Ports: []corev1.ServicePort{
			{
				Name: httpMetricsName, // http
				Port: kubeproxyPort,
			},
		},
		labelSelector: kubeproxyLabelSelector,
		Annotations: map[string]string{
			"prometheus.io/port":   strconv.FormatInt(int64(kubeproxyPort), 10),
			"prometheus.io/scrape": "true",
		},
	}

	p, err := NewPATCH(kubeConfigPath, apiserverHost, timeout, controller, scheduler, kubelet, kubeProxy)
	if err != nil {
		log.Fatalf("Error while initializing connection to Kubernetes apiserver. Reason: %s\n", err)
	}

	err = p.CollectionInfo()
	if err != nil {
		log.Fatalf("Error while collection cluster infos, %s", err)
	}

	err = p.PatchKube()
	if err != nil {
		log.Fatalf("Error while patching the management components, %s", err)
	}

	if EnableKubelet {
		err = p.PatchKubeletOrProxy(kubelet)
		if err != nil {
			log.Fatalf("Error while patching the kubelet, %s", err)
		}
	}

	if EnableKubeProxy {
		err = p.PatchKubeletOrProxy(kubeProxy)
		if err != nil {
			log.Fatalf("Error while patching the kube-proxy, %s", err)
		}
	}

	log.Println("See you next time!")
}

func NewPATCH(kubeConfigPath string, apiserverHost string, timeout *int64, controller, scheduler, kubelet, kubeproxy *svcInfo) (*PATCH, error) {
	var (
		config *rest.Config
		err    error
		p      = &PATCH{
			timeout:    timeout,
			controller: controller,
			scheduler:  scheduler,
			kubelet:    kubelet,
			kubeproxy:  kubeproxy,
			masterIPs:  make([]string, 0),
			nodeIPs:    make([]string, 0),
		}
	)

	if len(apiserverHost) > 0 || len(kubeConfigPath) > 0 {
		log.Println("Skipping in-cluster config")
		config, err = clientcmd.BuildConfigFromFlags(apiserverHost, kubeConfigPath)
		if err != nil {
			return nil, err
		}
	} else {
		config, err = rest.InClusterConfig()
		log.Println("Using in-cluster config to connect to apiserver")
		if err != nil {
			return nil, fmt.Errorf("the cluster may not configured: service account, or %s", err.Error())
		}
	}

	p.clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (p *PATCH) CollectionInfo() error {
	nodes, err := p.clientset.CoreV1().Nodes().List(metav1.ListOptions{
		TimeoutSeconds: p.timeout,
	})
	if err != nil {
		return err
	}
	for _, v := range nodes.Items {
		for _, v := range v.Status.Addresses {
			if v.Type == corev1.NodeInternalIP {
				p.nodeIPs = append(p.nodeIPs, v.Address)
			}
		}
	}

	kubePodLists, err := p.clientset.CoreV1().Pods(metav1.NamespaceSystem).List(metav1.ListOptions{
		LabelSelector:  "tier=control-plane,component=kube-apiserver",
		TimeoutSeconds: p.timeout,
	})
	if err != nil {
		return err
	}
	if len(kubePodLists.Items) != 0 { // kubeadm cluster
		log.Println("The cluster may be deployed by kubeadm")
		for _, v := range kubePodLists.Items {
			p.masterIPs = append(p.masterIPs, v.Status.HostIP)
		}
	} else { // managed by systemd
		log.Println("The cluster may be deployed by systemd")
		p.bin = true
		masterEPs, err := p.clientset.CoreV1().Endpoints(metav1.NamespaceDefault).Get("kubernetes", metav1.GetOptions{})
		if err != nil {
			return err
		}
		if len(masterEPs.Subsets) != 0 {
			for _, v := range masterEPs.Subsets[0].Addresses { //第一个端口,不需要其他端口
				p.masterIPs = append(p.masterIPs, v.IP)
			}
		} else {
			return fmt.Errorf("couldn't found the kubernetes master services at default namespaces")
		}
	}

	return nil
}

func (p *PATCH) PatchKube() error {
	//deal with the kube svc
	if err := p.patchSingleKubeSVC(p.controller); err != nil {
		return fmt.Errorf("cannot patching svc %s because %s", p.controller.svcName, err)
	}

	if err := p.patchSingleKubeSVC(p.scheduler); err != nil {
		return fmt.Errorf("cannot patching svc %s because %s", p.scheduler.svcName, err)
	}

	if p.bin { // systemd need to create or patch ep
		if err := p.patchKubeEP(p.controller); err != nil {
			return fmt.Errorf("patching the ep of the kube-controller-manager %s", err)
		}
		if err := p.patchKubeEP(p.scheduler); err != nil {
			return fmt.Errorf("patching the ep of the kube-scheduler %s", err)
		}
	} else {
		log.Println("The cluster may be deployed by kubeadm, don't need to patch the endpoints")
	}

	return nil
}

func (p *PATCH) patchSingleKubeSVC(info *svcInfo) error {
	_, err := p.clientset.CoreV1().Services(metav1.NamespaceSystem).Get(info.svcName, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if errors.IsNotFound(err) {
		_, err = p.clientset.CoreV1().Services(metav1.NamespaceSystem).Create(generateKubeSVC(p.bin, info))
		if err != nil {
			return err
		}
		log.Printf("Successfully created service %s at the ns %s\n", info.svcName, metav1.NamespaceSystem)
	}
	return nil
}

func generateKubeSVC(bin bool, info *svcInfo) *corev1.Service {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        info.svcName,
			Namespace:   metav1.NamespaceSystem,
			Labels:      info.labels,
			Annotations: info.Annotations,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Ports:     info.Ports,
		},
	}
	if !bin { //kubeadm staticPod
		svc.Spec.Selector = info.labelSelector
	}
	return svc
}

// ep always exist
func (p *PATCH) patchKubeEP(info *svcInfo) error {

	kubeEP, err := p.clientset.CoreV1().Endpoints(metav1.NamespaceSystem).Get(info.labelSelector["component"], metav1.GetOptions{})
	if err != nil {
		return err
	}
	// need patch ep
	if len(kubeEP.Subsets) == 0 || len(kubeEP.Subsets[0].Addresses) != len(p.masterIPs) {
		epIPs := make([]corev1.EndpointAddress, 0)
		for _, ip := range p.masterIPs {
			epIPs = append(epIPs, corev1.EndpointAddress{
				IP: ip,
			})
		}
		log.Printf("the ep %s need patched\n", info.labelSelector["component"])
		patchInfo := corev1.Endpoints{
			Subsets: []corev1.EndpointSubset{
				{
					Addresses: epIPs,
				},
			},
		}
		for _, v := range info.Ports {
			patchInfo.Subsets[0].Ports = append(patchInfo.Subsets[0].Ports, corev1.EndpointPort{
				Name: v.Name,
				Port: v.Port,
			})
		}

		patchBytes, err := json.Marshal(patchInfo)
		if err != nil {
			return err
		}
		_, err = p.clientset.CoreV1().Endpoints(metav1.NamespaceSystem).Patch(info.labelSelector["component"], types.MergePatchType, patchBytes)
		if err != nil {
			return err
		}
		log.Printf("Successfully patched the ep %s\n", info.labelSelector["component"])
	}
	return nil
}

func (p *PATCH) PatchKubeletOrProxy(instance *svcInfo) error {

	_, err := p.clientset.CoreV1().Services(metav1.NamespaceSystem).Get(instance.svcName, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if errors.IsNotFound(err) {
		log.Printf("cannot find the %s' svc, create it\n", instance.svcName)
		_, err := p.clientset.CoreV1().Services(metav1.NamespaceSystem).Create(
			generateKubeSVC(p.bin, instance))
		if err != nil {
			return err
		}
		log.Printf("Successfully created the %s's svc\n", instance.svcName)
	}

	getEPs, err := p.clientset.CoreV1().Endpoints(metav1.NamespaceSystem).Get(instance.svcName, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	epIPs := make([]corev1.EndpointAddress, 0)
	for _, ip := range p.nodeIPs {
		elems := corev1.EndpointAddress{
			IP: ip,
		}
		if instance.labels["k8s-app"] == kubeletName {
			elems.TargetRef = &corev1.ObjectReference{
				Kind: "Node",
			}
		}
		epIPs = append(epIPs, elems)
	}
	if errors.IsNotFound(err) {
		log.Printf("cannot find the %s' ep, create it\n", instance.svcName)
		CreateEP := &corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name:      instance.svcName,
				Namespace: metav1.NamespaceSystem,
				Labels:    instance.labels,
			},
			Subsets: []corev1.EndpointSubset{
				{
					Addresses: epIPs,
				},
			},
		}
		for _, v := range instance.Ports {
			CreateEP.Subsets[0].Ports = append(CreateEP.Subsets[0].Ports, corev1.EndpointPort{
				Name: v.Name,
				Port: v.Port,
			})
		}
		_, err := p.clientset.CoreV1().Endpoints(metav1.NamespaceSystem).Create(CreateEP)
		if err != nil {
			return err
		}
		log.Printf("Successfully created the %s's svc\n", instance.svcName)
	}

	// need patch ep
	if len(getEPs.Subsets) == 0 || len(getEPs.Subsets[0].Addresses) != len(p.nodeIPs) {
		log.Printf("the ep %s need patched\n", instance.svcName)
		patchInfo := corev1.Endpoints{
			Subsets: []corev1.EndpointSubset{
				{
					Addresses: epIPs,
				},
			},
		}
		for _, v := range instance.Ports {
			patchInfo.Subsets[0].Ports = append(patchInfo.Subsets[0].Ports, corev1.EndpointPort{
				Name: v.Name,
				Port: v.Port,
			})
		}
		patchBytes, err := json.Marshal(patchInfo)
		if err != nil {
			return err
		}
		_, err = p.clientset.CoreV1().Endpoints(metav1.NamespaceSystem).Patch(instance.svcName, types.MergePatchType, patchBytes)
		if err != nil {
			return err
		}
		log.Printf("Successfully patched the ep %s\n", instance.svcName)
	}

	return nil
}

