package cli

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/baotoq/shitcoin/internal/domain/wallet"
	"github.com/baotoq/shitcoin/internal/infrastructure/persistence/jsonfile"
)

// testnet spawns a multi-node local network with a single command.
// Node 0 is the seed node and auto-mines. Other nodes connect to node 0.
func (c *CLI) testnet(args []string) {
	fs := flag.NewFlagSet("testnet", flag.ExitOnError)
	nodes := fs.Int("nodes", 3, "Number of nodes to spawn")
	basePort := fs.Int("base-port", 3000, "Starting P2P port")
	baseHTTPPort := fs.Int("base-http-port", 8080, "Starting HTTP port for REST/WS")
	configPath := fs.String("config", "etc/shitcoin.yaml", "Config file path for child processes")
	fs.Parse(args)

	if *nodes < 1 {
		fmt.Println("Error: -nodes must be at least 1")
		os.Exit(1)
	}

	// Create wallet for node 0 (used for mining)
	node0DataDir := fmt.Sprintf("data/node-%d", *basePort)
	node0WalletPath := fmt.Sprintf("%s/wallets.json", node0DataDir)

	w, err := wallet.NewWallet()
	if err != nil {
		fmt.Printf("Error creating wallet for node 0: %v\n", err)
		os.Exit(1)
	}

	walletRepo, err := jsonfile.NewWalletRepo(node0WalletPath)
	if err != nil {
		fmt.Printf("Error opening wallet repo: %v\n", err)
		os.Exit(1)
	}
	if err := walletRepo.Save(w); err != nil {
		fmt.Printf("Error saving wallet: %v\n", err)
		os.Exit(1)
	}

	minerAddress := w.Address()

	// Set up context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down testnet...")
		cancel()
	}()

	// Spawn child processes
	var cmds []*exec.Cmd
	var wg sync.WaitGroup

	for i := range *nodes {
		port := *basePort + i
		httpPort := *baseHTTPPort + i
		datadir := fmt.Sprintf("data/node-%d", port)

		cmdArgs := []string{
			"-f", *configPath,
			"startnode",
			"-port", fmt.Sprintf("%d", port),
			"-datadir", datadir,
			"-http-port", fmt.Sprintf("%d", httpPort),
		}

		if i == 0 {
			// Node 0 is the miner / seed node
			cmdArgs = append(cmdArgs, "-mine", minerAddress)
		} else {
			// Other nodes connect to node 0 as seed peer
			cmdArgs = append(cmdArgs, "-peers", fmt.Sprintf("localhost:%d", *basePort))
		}

		cmd := exec.CommandContext(ctx, os.Args[0], cmdArgs...)
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		// Set up stdout/stderr prefixed pipes
		setupPrefixedOutput(&wg, cmd, i)

		cmds = append(cmds, cmd)
	}

	// Start all nodes
	for i, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			fmt.Printf("Error starting node %d: %v\n", i, err)
			cancel()
			break
		}
	}

	// Print status table
	fmt.Printf("\nTestnet started with %d nodes:\n", *nodes)
	for i := range *nodes {
		port := *basePort + i
		httpPort := *baseHTTPPort + i
		if i == 0 {
			fmt.Printf("  Node %d: P2P port %d, HTTP port %d, mining to %s (seed node)\n", i, port, httpPort, minerAddress)
		} else {
			fmt.Printf("  Node %d: P2P port %d, HTTP port %d, connected to localhost:%d\n", i, port, httpPort, *basePort)
		}
	}
	fmt.Println("Press Ctrl+C to stop all nodes.")
	fmt.Println()

	// Wait for all processes to exit
	exitCh := make(chan struct{})
	go func() {
		for i, cmd := range cmds {
			if cmd.Process == nil {
				continue
			}
			if err := cmd.Wait(); err != nil {
				// Only log unexpected exits (not from context cancellation)
				if ctx.Err() == nil {
					fmt.Printf("[node-%d] exited unexpectedly: %v\n", i, err)
				}
			}
		}
		close(exitCh)
	}()

	// Wait for either context cancellation or all processes to exit
	select {
	case <-ctx.Done():
		// Signal all child processes to terminate
		for i, cmd := range cmds {
			if cmd.Process == nil {
				continue
			}
			// Send SIGTERM to process group
			if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM); err != nil {
				fmt.Printf("[node-%d] failed to send SIGTERM: %v\n", i, err)
			}
		}

		// Wait with 5-second timeout, then SIGKILL
		done := make(chan struct{})
		go func() {
			wg.Wait()
			<-exitCh
			close(done)
		}()

		select {
		case <-done:
			fmt.Println("All nodes stopped.")
		case <-time.After(5 * time.Second):
			fmt.Println("Timeout waiting for nodes, sending SIGKILL...")
			for i, cmd := range cmds {
				if cmd.Process == nil {
					continue
				}
				if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
					// Process may have already exited
					_ = err
				}
				_ = i
			}
			<-exitCh
			fmt.Println("All nodes killed.")
		}
	case <-exitCh:
		// All processes exited on their own
		fmt.Println("All nodes exited.")
	}

	wg.Wait()
}

// setupPrefixedOutput creates pipes for stdout and stderr of a command,
// prefixing each line with [node-N] for readability.
func setupPrefixedOutput(wg *sync.WaitGroup, cmd *exec.Cmd, nodeIndex int) {
	prefix := fmt.Sprintf("[node-%d] ", nodeIndex)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Error creating stdout pipe for node %d: %v\n", nodeIndex, err)
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Printf("Error creating stderr pipe for node %d: %v\n", nodeIndex, err)
		return
	}

	wg.Add(2)
	go prefixLines(wg, stdout, prefix, os.Stdout)
	go prefixLines(wg, stderr, prefix, os.Stderr)
}

// prefixLines reads lines from r and writes them to w with the given prefix.
func prefixLines(wg *sync.WaitGroup, r io.ReadCloser, prefix string, w io.Writer) {
	defer wg.Done()
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fmt.Fprintf(w, "%s%s\n", prefix, scanner.Text())
	}
}
