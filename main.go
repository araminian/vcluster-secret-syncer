package main

import (
	"github.com/araminian/vcluster-secret-syncer/syncers"
	"github.com/loft-sh/vcluster-sdk/plugin"
)

func main() {
	// resolve configuration from environment variables

	ctx := plugin.MustInit()
	plugin.MustRegister(syncers.NewSecretSyncer(ctx))
	plugin.MustStart()
}
