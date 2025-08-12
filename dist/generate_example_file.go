package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

func main() {
	if err := do(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}

func do() error {
	rootOut := &strings.Builder{}
	rootCmd := exec.Command("go", "env", "GOROOT")
	rootCmd.Stdout = rootOut

	if err := rootCmd.Run(); err != nil {
		return fmt.Errorf("failed to retrieve GOROOT: %w", err)
	}

	const dirPerms = 0o755
	if err := os.MkdirAll("./build/dist", dirPerms); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	goRoot := strings.TrimSpace(rootOut.String())

	cmd := exec.Command("go", "run",
		filepath.Join(goRoot, "src", "crypto", "tls", "generate_cert.go"),
		"-ca",
		"-duration", "4380h",
		"-host", "127.0.0.1,::1,localhost")
	cmdOut := &strings.Builder{}
	cmd.Stdout = cmdOut
	cmd.Dir = "./build/dist"

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate certificate: %w: %s", err, cmdOut.String())
	}

	cert, err := os.ReadFile("./build/dist/cert.pem")
	if err != nil {
		return fmt.Errorf("failed to read certificate file: %w", err)
	}

	key, err := os.ReadFile("./build/dist/key.pem")
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	temp, err := os.ReadFile("./dist/example_template.yaml")
	if err != nil {
		return fmt.Errorf("failed to read template file: %w", err)
	}

	const indentSize = 5
	indent := strings.Repeat("  ", indentSize)
	certBlock := "|\n" + indent + strings.ReplaceAll(string(cert), "\n", "\n"+indent)
	keyBlock := "|\n" + indent + strings.ReplaceAll(string(key), "\n", "\n"+indent)
	certBlock = strings.TrimSpace(certBlock)
	keyBlock = strings.TrimSpace(keyBlock)

	data := map[string]any{
		// R66-TLS local server
		"r66Certificate": certBlock,
		"r66PrivateKey":  keyBlock,
		// HTTPS local server
		"httpsCertificate": certBlock,
		"httpsPrivateKey":  keyBlock,
		// FTPS local server
		"ftpsCertificate": certBlock,
		"ftpsPrivateKey":  keyBlock,
		// PeSIT-TLS local server
		"pesitCertificate": certBlock,
		"pesitPrivateKey":  keyBlock,
		// Waarp Transfer R66-TLS partner
		"wtCertificate": certBlock,
	}

	parser, err := template.New("conf_example").Parse(string(temp))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	outFile, err := os.Create("./build/dist/example_config.yaml")
	if err != nil {
		return fmt.Errorf("failed to create file output file: %w", err)
	}

	if err = parser.Execute(outFile, data); err != nil {
		defer func() {
			_ = outFile.Close()
			_ = os.Remove(outFile.Name())
		}()

		return fmt.Errorf("failed to execute template: %w", err)
	}

	if err = outFile.Close(); err != nil {
		return fmt.Errorf("failed to close output file: %w", err)
	}

	return nil
}
