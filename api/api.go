package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jnfrati/boquita/internal/logger"
	"github.com/jnfrati/boquita/internal/models"
	"github.com/jnfrati/boquita/pkg/controller"
)

func ReadBody[T any](r *http.Request) (*T, error) {

	bodyReader, err := r.GetBody()
	if err != nil {
		return nil, err
	}

	value := new(T)
	if err := json.NewDecoder(bodyReader).Decode(value); err != nil {
		return nil, err
	}

	return value, nil
}

func handleErr(ctx *gin.Context, err error) {
	logger.Global.Error().
		Str("method", ctx.Request.Method).
		Str("url.path", ctx.Request.URL.Path).
		Err(err).
		Msgf("error occured while processing the request")

	ctx.JSON(http.StatusInternalServerError, gin.H{
		"error": err.Error(),
	})
}

type ListJobsResponse struct {
	Jobs []models.Job `json:"jobs"`
}

func Start(ctx context.Context, controller *controller.Controller) error {

	r := gin.Default()

	r.GET("/v0/jobs", func(ctx *gin.Context) {
		jobs, err := controller.ListJobs(ctx)
		if err != nil {
			handleErr(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, jobs)
	})

	r.GET("/v0/jobs/:id", func(ctx *gin.Context) {
		jobId, ok := ctx.Params.Get("id")
		if !ok {
			err := errors.New("id must exist")
			handleErr(ctx, err)
		}
		job, err := controller.GetById(ctx, jobId)
		if err != nil {
			handleErr(ctx, err)
			return
		}
		ctx.JSON(http.StatusOK, job)
	})

	r.POST("/v0/jobs", func(ctx *gin.Context) {
		newJob := new(models.JobManifestV1)

		if err := ctx.ShouldBindBodyWithJSON(newJob); err != nil {
			handleErr(ctx, err)
			return
		}

		id, err := controller.CreateJob(ctx, newJob)
		if err != nil {
			handleErr(ctx, err)
			return
		}

		ctx.JSON(http.StatusCreated, gin.H{
			"job_id": id,
		})
	})

	srv := &http.Server{
		Addr:           "localhost:3333",
		Handler:        r,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// Start server in a goroutine
	go func() {
		logger.Global.Info().Msg("Starting API server on localhost:3333")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Global.Error().Err(err).Msg("API server failed to start")
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Create a deadline for shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger.Global.Info().Msg("Shutting down API server...")

	// Attempt graceful shutdown
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Global.Error().Err(err).Msg("API server forced to shutdown")
		return err
	}

	logger.Global.Info().Msg("API server gracefully stopped")
	return nil
}
