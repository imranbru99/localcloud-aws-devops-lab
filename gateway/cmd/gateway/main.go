package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/imrandev/localcloud/console"
	"github.com/imrandev/localcloud/gateway/pkg/router"
	_ "github.com/imrandev/localcloud/services/cloudwatch"
	_ "github.com/imrandev/localcloud/services/dynamodb"
	_ "github.com/imrandev/localcloud/services/ec2"
	_ "github.com/imrandev/localcloud/services/iam"
	_ "github.com/imrandev/localcloud/services/kms"
	_ "github.com/imrandev/localcloud/services/lambda"
	_ "github.com/imrandev/localcloud/services/s3"
	_ "github.com/imrandev/localcloud/services/secretsmanager"
	_ "github.com/imrandev/localcloud/services/sns"
	_ "github.com/imrandev/localcloud/services/sqs"
	_ "github.com/imrandev/localcloud/services/ssm"
	_ "github.com/imrandev/localcloud/services/sts"
)

func main() {
	port := os.Getenv("LOCALCLOUD_PORT")
	if port == "" {
		port = "4566"
	}

	consoleH := console.NewHandler()
	rt := router.NewRouter(consoleH)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: rt,
	}

	banner := `
======================================================================
  __                    _ _____ _                 _ 
 / /   ___   ___ __ _  | / ____| |               | |
/ /   / _ \ / __/ _` + "`" + ` | | | |    | | ___  _   _  __| |
\ \  | (_) | (_| (_| | | | |____| |/ _ \| | | |/ _` + "`" + ` |
 \_\  \___/ \___\__,_| |_|\_____|_|\___/ \__,_|\__,_|
                                                    
  AWS-Compatible Local Cloud Platform for Learners & Students
  Developer: Imran Ahmed (https://imrandev.bd)
======================================================================
  [✓] Edge Gateway Port:        http://127.0.0.1:%s
  [✓] AWS Management Console:   http://127.0.0.1:%s/console
  [✓] Live API Inspector:       http://127.0.0.1:%s/api/stream/inspector
  [✓] AWS CLI Endpoint URL:     --endpoint-url http://localhost:%s

  Quick Start in Terminal:
    export AWS_ACCESS_KEY_ID="test"
    export AWS_SECRET_ACCESS_KEY="test"
    export AWS_DEFAULT_REGION="us-east-1"
    aws --endpoint-url http://localhost:%s sts get-caller-identity
======================================================================
`
	fmt.Printf(banner, port, port, port, port, port)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Gateway server error: %v\n", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\nShutting down LocalCloud gateway...")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	fmt.Println("LocalCloud gateway stopped cleanly.")
}
