package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/OpenNHP/StealthDNS/cert"
	"github.com/OpenNHP/StealthDNS/dns"
	"github.com/urfave/cli/v2"
)

// Run the program as an administrator to automatically set up the DNS proxy.
// On Linux/MacOS, the program needs to be started with the root account or
// sudo command to ensure it can properly listen on port 53.
func main() {
	app := cli.NewApp()
	app.Name = "StealthDNS"
	app.Usage = "local DNS proxy service"
	app.Version = "0.1"

	app.Action = func(c *cli.Context) error {
		return runApp()
	}

	runCmd := &cli.Command{
		Name:    "run",
		Aliases: []string{"r"},
		Usage:   "create and run local DNS proxy service",
		Action: func(c *cli.Context) error {
			return runApp()
		},
	}

	certInstallCmd := &cli.Command{
		Name:    "install-root-ca",
		Usage:   "install root CA",
		Aliases: []string{"i"},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "ensure-file",
				Aliases: []string{"e"},
				Usage:   "if true, create the rootCA file if it does not exist; if false, fail or skip when missing.",
				Value:   true,
			},
		},
		Action: func(c *cli.Context) error {
			ensureFile := c.Bool("ensure-file")
			return cert.Install(ensureFile)
		},
	}

	certUninstallCmd := &cli.Command{
		Name:    "uninstall-root-ca",
		Usage:   "uninstall root CA",
		Aliases: []string{"u"},
		Action: func(c *cli.Context) error {
			return cert.Uninstall()
		},
	}

	certCreateCmd := &cli.Command{
		Name:    "create-cert",
		Aliases: []string{"create", "c"},
		Usage:   "create a certificate from a CSR file or a domain name.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "csr-file",
				Aliases: []string{"f"},
				Usage:   "create a certificate from a CSR file; specify the path to the CSR file.",
			},
			&cli.StringFlag{
				Name:    "domain-name",
				Aliases: []string{"d"},
				Usage:   "create a certificate using a domain name; please specify the domain name.",
			},
		},
		Action: func(c *cli.Context) error {
			csrFile := c.String("csr-file")
			domainName := c.String("domain-name")
			if csrFile == "" && domainName == "" {
				return fmt.Errorf("--csr-file and --domain-name cannot both be empty")
			}
			if csrFile != "" && domainName != "" {
				return fmt.Errorf("--csr-file and --domain-name are mutually exclusive; please specify only one")
			}
			return cert.CreateCert(csrFile, domainName)
		},
	}

	app.Commands = []*cli.Command{
		runCmd,
		certInstallCmd,
		certUninstallCmd,
		certCreateCmd,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(os.Stderr, err)
	}
}

func runApp() error {
	fmt.Println("Stealth DNS starting")
	exeFilePath, err := os.Executable()
	if err != nil {
		return err
	}
	exeDirPath := filepath.Dir(exeFilePath)

	p := &dns.ProxyService{}
	err = p.Start(exeDirPath, 4)
	if err != nil {
		return err
	}
	//react to terminate signals
	termCh := make(chan os.Signal, 1)
	signal.Notify(termCh, syscall.SIGTERM, os.Interrupt, syscall.SIGABRT)

	// block until terminated
	<-termCh
	p.Stop()
	return nil
}
