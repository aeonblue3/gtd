package commands

import (
	"flag"
	"fmt"
	"net/http"

	"gtd/internal/server"
	"gtd/internal/storage"
)

// Serve starts the optional HTTP API server.
func Serve(store storage.Backend, args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fs.String("addr", "127.0.0.1:8080", "Address to bind HTTP server")
	if err := fs.Parse(args); err != nil {
		return err
	}

	srv := server.New(store)
	fmt.Printf("Serving GTD API at http://%s\n", *addr)
	return http.ListenAndServe(*addr, srv.Handler())
}
