// vria-api serves the registry slice of contracts/21 on :8080.
// Local/dev entrypoint; production runs behind an OIDC-terminating gateway
// on Azure Container Apps (gate-c-runtime/08 ADR-01).
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/aaraminds/vria/internal/api"
	"github.com/aaraminds/vria/internal/registry"
)

func main() {
	addr := os.Getenv("VRIA_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	srv := api.NewServer(registry.NewMemStore())
	log.Printf("vria-api listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, srv))
}
