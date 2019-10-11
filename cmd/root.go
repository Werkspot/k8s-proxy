package cmd

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// VarnishPurgerCommand main application
type VarnishPurgerCommand struct {
	rootCommand *cobra.Command
	searcher    *VarnishPurger
}

// NewVarnishPurgerCommand returns a pointer to VarnishPurgerCommand
func NewVarnishPurgerCommand() *VarnishPurgerCommand {
	return &VarnishPurgerCommand{
		rootCommand: getRootCommand(),
	}
}

// Run the main application
func (v *VarnishPurgerCommand) Run() int {
	v.rootCommand.PersistentPreRunE = func(cmd *cobra.Command, args []string) (err error) {
		v.searcher = NewVarnishPurger(
			viper.GetString("kube_config"),
			viper.GetString("kube_context"),
			args[0],
			args[1],
			args[2:],
		)

		return v.searcher.init()
	}

	v.rootCommand.RunE = func(cmd *cobra.Command, args []string) (err error) {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "PURGE" {
				w.Header().Add("Allow", "PURGE")
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			if err = v.searcher.Purge(r.URL.Path); err != nil {
				log.Println(fmt.Printf("[ERROR] %s",
					err.Error(),
				))

				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})

		log.Println(fmt.Sprintf("Server started, listening in :%s", viper.GetString("port")))
		return http.ListenAndServe(fmt.Sprintf(":%s", viper.GetString("port")), nil)
	}

	if err := v.rootCommand.Execute(); err != nil {
		log.Println(fmt.Printf("[ERROR] %s",
			err.Error(),
		))
		return 1
	}

	return 0
}

func (v *VarnishPurgerCommand) populateConfig() (err error) {
	viper.AddConfigPath(".")

	viper.SetConfigName(".config")
	viper.SetEnvPrefix("VARNISH_PURGER")
	viper.AutomaticEnv()

	return viper.ReadInConfig()
}

func getRootCommand() (c *cobra.Command) {
	c = &cobra.Command{
		Use:           "varnish-purger [flags] namespace port-name label1 label2 label3",
		Short:         "Allows to purge all varnish instances living in a cluster",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	c.PersistentFlags().String("port", "8080", "HTTP port")
	c.PersistentFlags().String("kube-config", "", "Path to kubeconfig file")
	c.PersistentFlags().String("kube-context", "", "Context to use")

	flags := []string{
		"port",
		"kube-config",
		"kube-context",
	}

	for _, flag := range flags {
		if err := viper.BindPFlag(strings.ReplaceAll(flag, "-", "_"), c.PersistentFlags().Lookup(flag)); err != nil {
			panic(err)
		}
	}

	c.Args = cobra.MinimumNArgs(3)

	return
}
