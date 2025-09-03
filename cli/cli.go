package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"

	"github.com/jnfrati/boquita/api"
	"github.com/jnfrati/boquita/internal/executor"
	"github.com/jnfrati/boquita/internal/logger"
	"github.com/jnfrati/boquita/internal/models"
	"github.com/jnfrati/boquita/internal/queue"
	"github.com/jnfrati/boquita/internal/storage"
	"github.com/jnfrati/boquita/pkg/controller"
)

func main() {
	fmt.Println("hello")

	var rootCmd = &cobra.Command{
		Use:   "boquita",
		Short: "A simple CLI for managing boquita jobs",
		Long:  "A simple command-line tool to query and list jobs",
	}

	rootCmd.PersistentFlags().StringP("host", "", "http://localhost:3333", "Boquita server host:port")

	var startServer = &cobra.Command{
		Use:   "start",
		Short: "Start a boquita server",
		Run: func(cmd *cobra.Command, args []string) {

			rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			eg, ctx := errgroup.WithContext(rootCtx)

			jobStorage, err := storage.NewStorage[models.Job](storage.StorageType_Memory)
			if err != nil {
				panic(err)
			}
			cronToJobStorage, err := storage.NewStorage[models.CronToJob](storage.StorageType_Memory)
			if err != nil {
				panic(err)
			}
			executionStorage, err := storage.NewStorage[models.Execution](storage.StorageType_Memory)
			if err != nil {
				panic(err)
			}

			chanQueue := queue.NewChannelQueue[models.Job](uint8(100))

			executor, err := executor.NewExecutor(
				executor.ExecutorPlatform_UnikraftCloud,
				chanQueue.Client(),
				executionStorage,
			)
			if err != nil {
				panic(err)
			}

			controller := controller.NewController(
				chanQueue.Client(),
				jobStorage,
				cronToJobStorage,
				executionStorage,
			)

			eg.Go(func() error {
				return chanQueue.Start(ctx)
			})

			eg.Go(func() error {
				return executor.Start(ctx)
			})

			eg.Go(func() error {
				return api.Start(ctx, controller)
			})

			if err := eg.Wait(); err != nil {
				// Don't panic on context cancellation (normal shutdown)
				if err == context.Canceled {
					log.Println("Server stopped")
					return
				}
				log.Printf("Server error: %v", err)
				panic(err)
			}

			log.Println("Server shutdown complete")
			return
		},
	}

	var createJobCmd = &cobra.Command{
		Use:   "create [filepath]",
		Short: "Create a job",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			filepath := args[0]
			filepath = path.Clean(filepath)

			jobManifestYaml, err := os.ReadFile(filepath)
			if err != nil {
				log.Fatal(err.Error())
			}

			job := new(models.JobManifestV1)
			err = yaml.Unmarshal(jobManifestYaml, job)
			if err != nil {
				log.Fatal(err.Error())
			}

			log.Printf("%v", job)
			_, res, err := mutate[map[string]any](cmd, "/v0/jobs", job)
			if err != nil {
				log.Fatal(err.Error())
			}

			log.Printf("%v", res)
		},
	}

	// List command
	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List all jobs",
		Run: func(cmd *cobra.Command, args []string) {
			jobs, _, err := query[[]models.Job](cmd, "/v0/jobs")
			if err != nil {
				log.Fatalf("%v", err)
			}

			if len(jobs) == 0 {
				log.Println("No jobs found")
				return
			}
			logger.Global.Debug().Any("jobs", jobs).Msg("got jobs")

			log.Println("Jobs:")
			log.Println("-----")
			for _, job := range jobs {
				if job.LastExecution != nil {
					fmt.Printf("• %s (%s)\n  Status: %d\n\n", job.Manifest.Name, job.Id, job.LastExecution.Status)
				} else {
					fmt.Printf("• %s (%s)\n  Status: %s\n\n", job.Manifest.Name, job.Id, "not executed yet")
				}

			}
		},
	}

	// Add commands to root
	// rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(createJobCmd)
	rootCmd.AddCommand(startServer)

	// Execute the CLI
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func query[T any](cmd *cobra.Command, path string) (T, *http.Response, error) {
	host, _ := cmd.Flags().GetString("host")

	res, _ := http.Get(host + path)

	var obj T

	if err := json.NewDecoder(res.Body).Decode(&obj); err != nil {
		return obj, res, err
	}

	return obj, res, nil
}

func mutate[RT any](cmd *cobra.Command, path string, body any) (RT, *http.Response, error) {
	var obj RT

	host, _ := cmd.Flags().GetString("host")
	log.Printf("host %s", host)

	bodyJson := bytes.NewBuffer([]byte{})

	err := json.NewEncoder(bodyJson).Encode(body)
	if err != nil {
		return obj, nil, err
	}

	res, err := http.Post(host+path, "application/json", bodyJson)
	if err != nil {
		return obj, res, err
	}

	if err := json.NewDecoder(res.Body).Decode(&obj); err != nil {
		return obj, res, err
	}

	return obj, res, nil

}
