package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	emporousconfig "github.com/emporous/emporous-go/config"
	"github.com/emporous/emporous-go/content/layout"
	"github.com/emporous/emporous-go/model"
	"github.com/emporous/emporous-go/nodes/descriptor"
	"github.com/emporous/emporous-go/registryclient/orasclient"
	"github.com/emporous/emporous-go/util/examples"
	"github.com/spf13/cobra"
	"github.com/winfsp/cgofuse/fuse"

	"github.com/emporous-community/emporous-fuse-go/config"
	"github.com/emporous-community/emporous-fuse-go/fs"
)

var clientMountExamples = []examples.Example{
	{
		RootCommand:   filepath.Base(os.Args[0]),
		CommandString: "mount localhost:5001/test:latest ./mount-dir/",
		Descriptions: []string{
			"Mount collection reference.",
		},
	},
}

// MountOptions describe configuration options that can
// be set using the pull subcommand.
type MountOptions struct {
	*config.RootOptions
	Source         string
	MountPoint     string
	Insecure       bool
	PlainHTTP      bool
	Configs        []string
	AttributeQuery string
	NoVerify       bool
}

// NewMountCmd creates a new cobra.Command for the mount subcommand.
// TODO decide whether to use traditional mount -o flag format or to reuse emporous-go flags
func NewMountCmd(rootOpts *config.RootOptions) *cobra.Command {
	o := MountOptions{RootOptions: rootOpts}

	cmd := &cobra.Command{
		Use:           "mount [flags] SRC MOUNTPOINT",
		Short:         "Mount an Emporous collection based on content or attribute address",
		Example:       examples.FormatExamples(clientMountExamples...),
		SilenceErrors: false,
		SilenceUsage:  false,
		Args:          cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			cobra.CheckErr(o.Complete(args))
			cobra.CheckErr(o.Validate())
			cobra.CheckErr(o.Run(cmd.Context()))
		},
	}

	cmd.Flags().StringArrayVarP(&o.Configs, "configs", "c", o.Configs, "auth config paths when contacting registries")
	cmd.Flags().BoolVarP(&o.Insecure, "insecure", "i", o.Insecure, "allow connections to SSL registry without certs")
	cmd.Flags().BoolVar(&o.PlainHTTP, "plain-http", o.PlainHTTP, "use plain http and not https when contacting registries")
	cmd.Flags().StringVarP(&o.MountPoint, "output", "o", o.MountPoint, "output location for artifacts")
	cmd.Flags().StringVar(&o.AttributeQuery, "attributes", o.AttributeQuery, "attribute query config path")
	cmd.Flags().BoolVarP(&o.NoVerify, "no-verify", "", o.NoVerify, "skip collection signature verification")

	return cmd
}

func (o *MountOptions) Complete(args []string) error {
	if len(args) < 2 {
		return errors.New("bug: expecting one argument")
	}
	o.Source = args[0]
	o.MountPoint = args[1]
	return nil
}

func (o *MountOptions) Validate() error {
	mountPointStat, err := os.Stat(o.MountPoint)
	if err != nil {
		return err
	}
	if !mountPointStat.IsDir() {
		return errors.New("mount point must be a directory")
	}
	return nil
}

func unmountOnInterrupt(host *fuse.FileSystemHost) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(
		interrupt,
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	<-interrupt
	host.Unmount()
}

func (o *MountOptions) Run(ctx context.Context) error {
	cache, err := layout.NewWithContext(ctx, o.CacheDir)
	if err != nil {
		return err
	}

	var clientOpts = []orasclient.ClientOption{
		orasclient.SkipTLSVerify(o.Insecure),
		orasclient.WithAuthConfigs(o.Configs),
		orasclient.WithPlainHTTP(o.PlainHTTP),
		orasclient.WithCache(cache),
	}

	o.Logger.Infof("Resolving artifacts for reference %s", o.Source)

	var matcher model.Matcher
	if o.AttributeQuery != "" {
		query, err := emporousconfig.ReadAttributeQuery(o.AttributeQuery)
		if err != nil {
			return err
		}

		matcher = descriptor.JSONSubsetMatcher(query.Attributes)
		clientOpts = append(clientOpts, orasclient.WithPullableAttributes(matcher))
	}

	if !o.NoVerify {
		o.Logger.Infof("Checking signature of %s", o.Source)
		//if err := verifyCollection(ctx, o.Source, o.Remote); err != nil {
		//	return err
		//}
	}

	client, err := orasclient.NewClient(clientOpts...)
	if err != nil {
		return fmt.Errorf("error configuring client: %v", err)
	}

	fuseHost := fuse.NewFileSystemHost(fs.NewEmporousFs(ctx, fs.EmporousFsOptions(*o), client, matcher))
	fuseHost.SetCapReaddirPlus(true)
	go unmountOnInterrupt(fuseHost)
	o.Logger.Infof("Mounting emporous to directory %v", o.MountPoint)
	opts := []string{
		"-o", "fsname=emporousfs",
		"-o", "ro",
		"-o", "default_permissions",
		"-o", "auto_unmount",
		//"-o", "user_xattr",
	}
	mounted := fuseHost.Mount(o.MountPoint, opts)
	o.Logger.Infof("%v", mounted)

	return nil
}
