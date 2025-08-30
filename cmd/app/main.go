package main

func main() {
	PrintVersion()

	cfg := loadConfig()
	logger := initLogger()
	defer logger.Sync()

	peerStore, nodeID, internalAddr := initBootstrap(cfg, logger)
	reg := initRegistry(cfg, logger)
	checker := initHealthChecker(cfg, reg, logger)

	runInitialHealth(reg, checker, logger)
	startHealthLoop(reg, checker, logger)

	registerRoutes(reg, checker, peerStore, nodeID, internalAddr, cfg, logger)

	startServer(cfg.Host, cfg.Port, logger)
}
