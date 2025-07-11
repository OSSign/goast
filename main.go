package main

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/clysec/greq"
)

const AZURESIGNTOOL_URL = "https://github.com/vcsjones/AzureSignTool/releases/latest/download/AzureSignTool-x64.exe"
const AZURESIGNTOOL_ARM_URL = "https://github.com/vcsjones/AzureSignTool/releases/latest/download/AzureSignTool-arm64.exe"

var NeedEnv = map[string]string{
	"AST_VAULT":     "-kvu",
	"AST_CERT":      "-kvc",
	"AST_IDENT":     "-kvi",
	"AST_SECRET":    "-kvs",
	"AST_TD":        "-td",
	"AST_TENANT":    "--azure-key-vault-tenant-id",
	"AST_TIMESTAMP": "-tr",
}

func main() {
	astUrl := AZURESIGNTOOL_URL
	astBinName := "AzureSignTool.exe"

	if runtime.GOARCH == "arm64" || runtime.GOARCH == "arm" {
		astUrl = AZURESIGNTOOL_ARM_URL
		astBinName = "AzureSignToolArm.exe"
	}

	astBinPath := filepath.Join(os.TempDir(), astBinName)

	if _, err := os.Stat(astBinPath); os.IsNotExist(err) {
		fmt.Println("AzureSignTool not found, downloading...")
		fileif, err := os.OpenFile(astBinPath, os.O_CREATE|os.O_WRONLY, 0755)
		if err != nil {
			panic(err)
		}
		defer fileif.Close()

		filereq := greq.GetRequest(astUrl).WithHeader("User-Agent", "AzureSignToolInstaller/1.0")
		resp, err := filereq.Execute()
		if err != nil {
			panic(err)
		}

		if resp.StatusCode != 200 {
			panic("Failed to download AzureSignTool: " + resp.Response.Status)
		}

		reader, err := resp.BodyReader()
		if err != nil {
			panic(err)
		}

		if nBytes, err := io.Copy(fileif, *reader); err != nil {
			panic(err)
		} else {
			fmt.Printf("Downloaded AzureSignTool (%d bytes) to %s", nBytes, astBinPath)
		}
	}

	fmt.Println("AzureSignTool is ready at:", astBinPath)

	var envs []string = []string{
		"sign",
	}

	for k, v := range NeedEnv {
		if val, ok := os.LookupEnv(k); ok {
			envs = append(envs, v, val)
		} else {
			newName := strings.TrimPrefix(k, "AST_")
			if val, ok := os.LookupEnv(fmt.Sprintf("AZURESIGNTOOL_%s", newName)); ok {
				envs = append(envs, v, val)
			} else {
				fmt.Fprintf(os.Stderr, "Environment variable %s is required but not set.\n", k)
				os.Exit(1)
			}
		}
	}

	for _, arg := range os.Args[1:] {
		if _, err := os.Stat(arg); err != nil {
			fmt.Fprintf(os.Stderr, "File %s does not exist (%v), skipping...\n", arg, err)
			os.Exit(1)
		}

		cmd := exec.Command(astBinPath, append(envs, arg)...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running AzureSignTool: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully signed %s\n", arg)
	}

	fmt.Println("All files signed successfully.")
}
