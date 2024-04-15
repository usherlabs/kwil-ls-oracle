package main

import (
	"fmt"
	"os"

	"github.com/kwilteam/kwil-db/cmd/kwild/root"
	_ "github.com/usherlabs/kwil-ls-oracle/internal/extensions/listeners/logstore_listener"
	_ "github.com/usherlabs/kwil-ls-oracle/internal/extensions/resolutions/ingest_resolution"
)

func main() {
	if err := root.RootCmd().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}

func init() {
}
