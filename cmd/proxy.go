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

// Proxy will allow us to purge all varnish living in a cluster
type Proxy struct {
	kubeConfig, kubeContext string
	clientSet               *kubernetes.Clientset
	namespaces, labels      []string
	portNames               map[string]bool
	timeout                 time.Duration
}

// NewProxy returns a pointer to Proxy
func NewProxy(kubeConfig, kubeContext string, namespaces, portNames, labels []string, timeout time.Duration) *Proxy {
	mapPortNames := make(map[string]bool)
	for _, s := range portNames {
		mapPortNames[s] = true
	}

	return &Proxy{
		kubeConfig:  kubeConfig,
		kubeContext: kubeContext,
		namespaces:  namespaces,
		portNames:   mapPortNames,
		labels:      labels,
		timeout:     timeout,
	}
}

// Forward request to all matching endpoints
func (p *Proxy) Forward(request *http.Request) (err error) {
	endpoints, err := p.getEndpoints()
	if err != nil {
		return
	}

	if len(endpoints) == 0 {
		return errors.New("no endpoints found")
	}

	cErr := make(chan error, len(endpoints))
	for _, endpoint := range endpoints {
		go func(request *http.Request, endpoint string) {
			cErr <- p.request(request, endpoint)
		}(request, endpoint)
	}

	var errs []error
	for range endpoints {
		select {
		case e := <-cErr:
			errs = append(errs, e)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%d/%d requests failed, check logs for more information", len(errs), len(endpoints))
	}

	return
}

func (p *Proxy) request(request *http.Request, host string) (err error) {
	defer func() {
		if err != nil {
			log.Println(fmt.Sprintf(
				"%s errored with: %s",
				host,
				err.Error(),
			))
		}
	}()

	client := &http.Client{
		Timeout: p.timeout,
	}

	req, err := http.NewRequest(
		request.Method,
		fmt.Sprintf("http://%s/%s", host, request.URL.RequestURI()),
		request.Body,
	)
	if err != nil {
		return
	}

	req.Header = request.Header

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	log.Println(fmt.Sprintf("%s, status: %d, body: %s", request.URL.String(), resp.StatusCode, body))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expecting %d, found: %d", http.StatusOK, resp.StatusCode)
	}

	return
}

func (p *Proxy) getEndpoints() (endpoints []string, err error) {
	for _, namespace := range p.namespaces {
		list, err := p.clientSet.CoreV1().Endpoints(namespace).List(
			metav1.ListOptions{
				LabelSelector: strings.Join(p.labels, ","),
			})

		if err != nil {
			return nil, err
		}

		for _, endpoint := range list.Items {
			for _, subset := range endpoint.Subsets {
				for _, port := range subset.Ports {
					if _, ok := p.portNames[port.Name]; !ok {
						continue
					}

					for _, address := range subset.Addresses {
						ep := fmt.Sprintf("%s:%s", address.IP, strconv.Itoa(int(port.Port)))
						endpoints = append(endpoints, ep)

						log.Println(fmt.Sprintf(
							"%s endpoint found in namespace %s with labels %s",
							ep,
							namespace,
							strings.Join(p.labels, ","),
						))
					}
				}
			}
		}
	}

	log.Println(fmt.Sprintf(
		"%d endpoints found in namespaces %s with labels %s",
		len(endpoints),
		strings.Join(p.namespaces, ","),
		strings.Join(p.labels, ","),
	))

	return
}

func (p *Proxy) init() (err error) {
	config, err := p.getKubeConfig()
	if err != nil {
		return
	}

	p.clientSet, err = kubernetes.NewForConfig(config)
	if err != nil {
		return
	}

	return p.checkConn()
}

func (p *Proxy) checkConn() (err error) {
	_, err = p.getEndpoints()
	return
}

func (p *Proxy) getKubeConfig() (config *rest.Config, err error) {
	if p.kubeConfig == "" {
		return rest.InClusterConfig()
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: p.kubeConfig},
		&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{}, CurrentContext: p.kubeContext},
	).ClientConfig()
}
