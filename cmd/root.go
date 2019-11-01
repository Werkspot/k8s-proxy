package cmd

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// K8sProxy main application
type K8sProxy struct {
	rootCommand    *cobra.Command
	proxy          *Proxy
	allowedMethods map[string]bool
}

// NewProxyCommand returns a pointer to K8sProxy
func NewProxyCommand() *K8sProxy {
	return &K8sProxy{
		rootCommand:    getRootCommand(),
		allowedMethods: make(map[string]bool),
	}
}

// Run the main application
func (k *K8sProxy) Run() int {
	k.rootCommand.Flags().StringSliceP("namespace", "n", nil, "K8s namespace")
	k.rootCommand.Flags().StringSliceP("label", "l", nil, "K8s endpoint matching label")
	k.rootCommand.Flags().StringSliceP("port-name", "p", nil, "K8s endpoint matching port name")
	k.rootCommand.Flags().Duration("timeout", 5*time.Second, "Proxy timeout")

	k.rootCommand.PersistentPreRunE = func(cmd *cobra.Command, args []string) (err error) {
		k.rootCommand.Flags().VisitAll(bindFlags)

		k.proxy = NewProxy(
			viper.GetString("kube_config"),
			viper.GetString("kube_context"),
			viper.GetStringSlice("namespace"),
			viper.GetStringSlice("port_name"),
			viper.GetStringSlice("label"),
			viper.GetDuration("timeout"),
		)

		for _, s := range viper.GetStringSlice("method") {
			k.allowedMethods[s] = true
		}

		return k.proxy.init()
	}

	k.rootCommand.RunE = func(cmd *cobra.Command, args []string) (err error) {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if _, ok := k.allowedMethods[r.Method]; !ok {
				w.Header().Add("Allow", strings.Join(viper.GetStringSlice("method"), ", "))
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			if err = k.proxy.Forward(r); err != nil {
				k.handleError(err)

				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})

		log.Println(fmt.Sprintf("Server started, listening in :%s", viper.GetString("http_port")))
		return http.ListenAndServe(fmt.Sprintf(":%s", viper.GetString("http_port")), nil)
	}

	if err := k.rootCommand.Execute(); err != nil {
		k.handleError(err)
		return 1
	}

	return 0
}

func (k *K8sProxy) populateConfig() (err error) {
	viper.AddConfigPath(".")

	viper.SetConfigName(".config")
	viper.SetEnvPrefix("K8S_PROXY")
	viper.AutomaticEnv()

	return viper.ReadInConfig()
}

func (k *K8sProxy) handleError(err error) {
	log.Println(fmt.Sprintf("[ERROR] %s",
		err.Error(),
	))
}

func bindFlags(flag *pflag.Flag) {
	if err := viper.BindPFlag(strings.ReplaceAll(flag.Name, "-", "_"), flag); err != nil {
		panic(err)
	}
}

func getRootCommand() (c *cobra.Command) {
	c = &cobra.Command{
		Use:           "k8s-proxy",
		Short:         "Allows to proxy an HTTP request to all matching endpoints",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	c.PersistentFlags().String("http-port", "8080", "HTTP port")
	c.PersistentFlags().String("kube-config", "", "Path to kubeconfig file")
	c.PersistentFlags().String("kube-context", "", "Context to use")
	c.PersistentFlags().StringSlice("method", nil, "HTTP methods allowed")

	return
}
