package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
)

func BuildRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "loadtest",
	}
	cmd.AddCommand(BuildServerCmd())
	cmd.AddCommand(BuildClientCmd())
	return cmd
}

func BuildServerCmd() *cobra.Command {
	return &cobra.Command{
		Use: "server",
		RunE: func(cmd *cobra.Command, args []string) error {
			router := mux.NewRouter()
			latConfig := LatencyConfig{
				Wait: time.Millisecond * 10,
			}
			s := NewServer(
				router,
				latConfig,
			)

			return s.ListenAndServe(cmd.Context())
		},
	}
}

func BuildClientCmd() *cobra.Command {
	var (
		addr        string
		concurrency int
		delay       time.Duration
	)
	cmd := &cobra.Command{
		Use: "client",
		RunE: func(cmd *cobra.Command, args []string) error {
			auth := os.Getenv("AUTH")
			client := NewClient(auth, WorkloadConfig{
				ConcurrentRequests: concurrency,
				Delay:              delay,
			})
			client.runHttp(cmd.Context(), addr)
			return nil
		},
	}
	cmd.Flags().StringVarP(
		&addr, "addr", "a", "http://localhost:8080", "address of remote server",
	)
	cmd.Flags().IntVarP(
		&concurrency,
		"concurrency",
		"c",
		100,
		"maximum number of concurrent in flight requests",
	)

	cmd.Flags().DurationVarP(
		&delay,
		"wait",
		"w",
		time.Millisecond,
		"time to wait between scheduling more requests",
	)

	return cmd
}

func main() {
	rootCmd := BuildRootCmd()

	if err := rootCmd.Execute(); err != nil {
		log.Println(fmt.Errorf("failted to execute root cmd : %s", err))
	}
}
