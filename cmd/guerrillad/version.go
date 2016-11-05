package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"

	guerrilla "github.com/flashmob/go-guerrilla"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version info",
	Long:  `All software has versions. This is Guerrilla's`,
	Run: func(cmd *cobra.Command, args []string) {
		logVersion()
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func logVersion() {
	log.Infof("guerrillad %s (%s)",
		guerrilla.Version, guerrilla.BuildTime)
}
