package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/pinebit/go-boot/boot"
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
		BaseContext: func(net.Listener) context.Context {
			return rootCtx
		},
	}

	services := boot.Sequentially(NewHttpServer(httpServer, logger))
	if err := services.Start(rootCtx); err != nil {
		cancel()
	}

	<-rootCtx.Done()
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := services.Stop(stopCtx); err != nil {
		logger.Errorw("failed to stop server gracefully", "err", err)
	} else {
		logger.Info("HTTP server is stopped gracefully")
	}
}
