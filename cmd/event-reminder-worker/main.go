package main

import (
	"bot/telegram/config"
	"bot/telegram/services"
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/go-co-op/gocron/v2"
)

func main() {
	if err := config.Init(); err != nil {
		fmt.Printf("Failed to load environment configuration: %s\n", err)
		return
	}

	if err := config.Current.ValidateBot(); err != nil {
		fmt.Println(err.Error())
		return
	}

	scheduler, err := gocron.NewScheduler()
	if err != nil {
		fmt.Printf("Failed to create scheduler: %s\n", err)
		return
	}

	var running atomic.Bool
	run := func() {
		if !running.CompareAndSwap(false, true) {
			fmt.Println("Reminder worker run skipped because previous run is still active")
			return
		}
		defer running.Store(false)

		if err := processDueReminders(); err != nil {
			fmt.Printf("Reminder worker run failed: %s\n", err)
		}
	}

	if _, err := scheduler.NewJob(gocron.DurationJob(time.Minute), gocron.NewTask(run)); err != nil {
		fmt.Printf("Failed to schedule reminder worker: %s\n", err)
		return
	}

	run()
	scheduler.Start()
	fmt.Println("Event reminder worker started")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	if err := scheduler.Shutdown(); err != nil {
		fmt.Printf("Failed to stop scheduler cleanly: %s\n", err)
	}
}

func processDueReminders() error {
	conn, err := services.GlobalPoolManager.GetConnectionFromPool(config.Current.DBName)
	if err != nil {
		return fmt.Errorf("get database connection: %w", err)
	}
	defer conn.Release()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()

	return services.ProcessDueEventReminders(ctx, conn.Conn())
}
