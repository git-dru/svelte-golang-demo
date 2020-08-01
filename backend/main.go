package main

//go:generate sqlboiler --wipe psql

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/davidk81/svelte-golang-demo/backend/patient"
	"github.com/davidk81/svelte-golang-demo/backend/patientdb"
	"github.com/davidk81/svelte-golang-demo/backend/session"
	_ "github.com/lib/pq"
	"github.com/valyala/fasthttp"
)

var (
	addr     = flag.String("addr", ":8000", "TCP address to listen to")
	compress = flag.Bool("compress", false, "Whether to enable transparent response compression")
)

func main() {
	flag.Parse()

	// init db
	patientdb.Init()

	h := requestHandler
	if *compress {
		h = fasthttp.CompressHandler(h)
	}

	go func() {
		log.Println("server starting on port", *addr)
		if err := fasthttp.ListenAndServe(*addr, h); err != nil {
			log.Fatalf("Error in ListenAndServe: %s", err)
		}
	}()

	// cleanup
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		_ = <-sigs
		done <- true
	}()
	<-done
	patientdb.Close()
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	log.Printf("%s %s\n", ctx.Request.Header.Method(), ctx.Path())

	ctx.Response.Header.Set("access-control-allow-credentials", "true")
	ctx.Response.Header.Set("access-control-allow-origin", string(ctx.Request.Header.Peek("Origin")))
	ctx.Response.Header.Set("access-control-expose-headers", "WWW-Authenticate,Server-Authorization")
	ctx.Response.Header.Set("cache-control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")

	switch string(ctx.Request.Header.Method()) {
	case "OPTIONS":
		handleMethodOptions(ctx)
		return
	}

	switch string(ctx.Path()) {
	case "/api/v1/session":
		session.HandleSession(ctx)
	case "/api/v1/patients":
		patient.HandlePatientList(ctx)
	case "/api/v1/patient":
		patient.HandlePatient(ctx)
	case "/api/v1/patient/note":
		patient.HandlePatientNote(ctx)
	default:
		ctx.Error("Unsupported path", fasthttp.StatusNotFound)
	}

	session.VerifySession(ctx, "nurse")

}

func handleMethodOptions(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.Set("access-control-allow-headers", "Accept,Authorization,Content-Type,If-None-Match")
	ctx.Response.Header.Set("access-control-allow-methods", string(ctx.Request.Header.Peek("Access-Control-Request-Method")))
	ctx.Response.Header.Set("access-control-max-age", "86400")
	ctx.SetStatusCode(fasthttp.StatusOK)
}