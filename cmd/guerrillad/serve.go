package main

import "github.com/spf13/cobra"

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "start the small SMTP server",
	Run:   serve,
}

var (
	iface string
)

func init() {
	serveCmd.PersistentFlags().StringVarP(&iface, "if", "", "",
		"Interface and port to listen on, eg. 127.0.0.1:2525 ")

	rootCmd.AddCommand(serveCmd)
}

func serve(cmd *cobra.Command, args []string) {
	// readConfig()
	// initialise()
	// if err := testDbConnections(); err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(1)
	// }
	// // start some savemail workers
	// for i := 0; i < mainConfig.Save_workers_size; i++ {
	// 	go saveMail()
	// }
	// // run our servers
	// for serverId := 0; serverId < len(mainConfig.Servers); serverId++ {
	// 	if mainConfig.Servers[serverId].Is_enabled {
	// 		go runServer(mainConfig.Servers[serverId])
	// 	}
	// }
	// sigHandler()
}
