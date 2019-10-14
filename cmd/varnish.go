package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	// Load oidc auth library
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

// VarnishPurger will allow us to purge all varnish living in a cluster
type VarnishPurger struct {
	kubeConfig, kubeContext, namespace, portName string
	clientSet                                    *kubernetes.Clientset
	labels                                       []string
}

// NewVarnishPurger returns a pointer to VarnishPurger
func NewVarnishPurger(kubeConfig, kubeContext, namespace, portName string, labels []string) *VarnishPurger {
	return &VarnishPurger{
		kubeConfig:  kubeConfig,
		kubeContext: kubeContext,
		namespace:   namespace,
		portName:    portName,
		labels:      labels,
	}
}

// Purge all matching varnish instances matching that url
func (v *VarnishPurger) Purge(url string) (err error) {
	endpoints, err := v.getEndPoints()
	if err != nil {
		return
	}

	if len(endpoints) == 0 {
		return errors.New("no endpoints found")
	}

	log.Println(fmt.Sprintf(
		"%d endpoints found in namespace %s with labels %s",
		len(endpoints),
		v.namespace,
		strings.Join(v.labels, ","),
	))

	for _, endpoint := range endpoints {
		err = v.purge(fmt.Sprintf("http://%s%s", endpoint, url))
		if err != nil {
			return
		}

		log.Println(fmt.Sprintf("endpoint %s purged", endpoint))
	}

	return
}

func (v *VarnishPurger) purge(url string) (err error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest("PURGE", url, nil)
	if err != nil {
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	log.Println(fmt.Sprintf("%s, status: %d, body: %s", url, resp.StatusCode, body))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("purge failed for %s", url)
	}

	return
}

func (v *VarnishPurger) getEndPoints() (endpoints []string, err error) {
	list, err := v.clientSet.CoreV1().Endpoints(v.namespace).List(metav1.ListOptions{LabelSelector: strings.Join(v.labels, ",")})
	if err != nil {
		return
	}

	if len(list.Items) == 0 {
		err = errors.New("no varnish instances found")
		return
	}

	var vPort string
	for _, endpoint := range list.Items {
		for _, subset := range endpoint.Subsets {
			for _, port := range subset.Ports {
				if port.Name == v.portName {
					vPort = strconv.Itoa(int(port.Port))
				}
			}
			if vPort == "" {
				continue
			}

			for _, address := range subset.Addresses {
				endpoints = append(endpoints, fmt.Sprintf("%s:%s", address.IP, vPort))
			}

			vPort = ""
		}
	}

	return
}

func (v *VarnishPurger) init() (err error) {
	config, err := v.getKubeConfig()
	if err != nil {
		return
	}

	v.clientSet, err = kubernetes.NewForConfig(config)
	if err != nil {
		return
	}

	return v.checkConn()
}

func (v *VarnishPurger) checkConn() (err error) {
	_, err = v.clientSet.CoreV1().Endpoints(v.namespace).List(metav1.ListOptions{})

	return
}

func (v *VarnishPurger) getKubeConfig() (config *rest.Config, err error) {
	if v.kubeConfig == "" {
		return rest.InClusterConfig()
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: v.kubeConfig},
		&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{}, CurrentContext: v.kubeContext},
	).ClientConfig()
}
