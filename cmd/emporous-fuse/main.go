package main

import (
	"github.com/spf13/cobra"

	"github.com/emporous-community/emporous-fuse-go/cmd/emporous-fuse/commands"
)

func main() {
	rootCmd := commands.NewRootCmd()
	cobra.CheckErr(rootCmd.Execute())
}
