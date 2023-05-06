package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"go.uber.org/zap"
)

func shutdownHandler(cancelFunc func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	<-c
	cancelFunc()
}

func main() {
	portFlag := flag.Int("port", 3000, "specifies a desired port number, default is 3000")
	headersFlag := flag.Bool("headers", false, "print request headers, default is false")
	flag.Parse()

	// Change to zap.NewProduction() to enable structured logging
	zapLogger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer zapLogger.Sync()

	logger := zapLogger.Sugar()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Errorw("failed to read request body", "err", err)
		}
		if len(body) == 0 {
			logger.Infof("%s %s", r.Method, r.URL.String())
		} else {
			logger.Infof("%s %s: %s", r.Method, r.URL.String(), string(body))
		}
		if *headersFlag {
			for k, v := range r.Header {
				logger.Infof("* %s: %s", k, strings.Join(v, ", "))
			}
		}
		w.WriteHeader(http.StatusOK)
	})

	rootCtx, cancel := context.WithCancel(context.Background())
	go shutdownHandler(cancel)

	httpServer := &http.Server{
		Addr: fmt.Sprintf(":%d", *portFlag),
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		logger.Infof("Listening on port: %s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalw("HTTP server failed to listen", "err", err)
		}
	}()

	go func() {
		defer wg.Done()

		<-rootCtx.Done()
		if err := httpServer.Shutdown(context.Background()); err != nil {
			logger.Errorw("HTTP server error", "err", err)
		}
	}()

	// Wait for graceful server shutdown
	wg.Wait()
	logger.Info("HTTP server is stopped gracefully")
}
