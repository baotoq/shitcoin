package main

import (
	"flag"

	"github.com/baotoq/shitcoin/internal/config"
	"github.com/baotoq/shitcoin/internal/handler/cli"
	"github.com/baotoq/shitcoin/internal/svc"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stat"
)

func main() {
	configFile := flag.String("f", "etc/shitcoin.yaml", "the config file")
	flag.Parse()

	// Suppress go-zero framework noise for clean CLI output
	logx.Disable()
	stat.DisableLog()

	// Load configuration
	var c config.Config
	conf.MustLoad(*configFile, &c)
	c.Consensus.ApplyDefaults()

	// Create service context (opens DB, wires dependencies)
	serviceCtx := svc.NewServiceContext(c)
	defer serviceCtx.Close()

	// Dispatch to CLI
	app := cli.New(serviceCtx)
	app.Run(flag.Args())
}
