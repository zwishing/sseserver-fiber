package main

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/zwishing/sseserver-fiber"
)

func main() {
	app := fiber.New()
	sse := sseserver.New()
	defer sse.Close()

	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{"Cache-Control"},
	}))

	app.Get("/sse", sse.Handler("progress"))

	go func() {
		ticker := time.NewTicker(1000 * time.Millisecond)
		defer ticker.Stop()

		for i := 1; i <= 100; i++ {
			<-ticker.C
			if err := sse.PublishEvent("progress", "processing-percent", []byte(fmt.Sprintf("%d%%", i))); err != nil {
				return
			}
		}
	}()

	if err := app.Listen(":8080"); err != nil {
		panic(err)
	}
}
