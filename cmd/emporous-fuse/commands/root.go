package commands

import (
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/emporous-community/emporous-fuse-go/cmd/emporous-fuse/commands/log"
	"github.com/emporous-community/emporous-fuse-go/config"
)

// NewRootCmd creates a new cobra.Command for the command root.
func NewRootCmd() *cobra.Command {
	o := config.RootOptions{}

	o.IOStreams = genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
	o.EnvConfig = config.ReadEnvConfig()
	cmd := &cobra.Command{
		Use:   filepath.Base(os.Args[0]),
		Short: "Emporous FUSE Driver",
		//Long:          clientLong,
		SilenceErrors: false,
		SilenceUsage:  false,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			logger, err := log.NewLogger(o.IOStreams.Out, o.LogLevel)
			if err != nil {
				return err
			}
			o.Logger = logger

			cacheEnv := os.Getenv("EMPOROUS_CACHE")
			if cacheEnv != "" {
				o.CacheDir = cacheEnv
			} else {
				// ~/.cache/emporous
				o.CacheDir = filepath.Join(xdg.CacheHome, "emporous")
			}

			return os.MkdirAll(o.CacheDir, 0750)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	f := cmd.PersistentFlags()
	f.StringVarP(&o.LogLevel, "loglevel", "l", "info",
		"Log level (debug, info, warn, error, fatal)")

	cmd.AddCommand(NewMountCmd(&o))
	cmd.AddCommand(NewVersionCmd(&o))

	return cmd
}
