// Command whatsapp reads a WhatsApp Business Account: numbers, templates, analytics.
package main

import (
	"os"

	"github.com/piyush-gambhir/whatsapp-cli/cli-go/cmd"
)

func main() {
	os.Exit(cmd.Execute(os.Stdin, os.Stdout, os.Stderr))
}
