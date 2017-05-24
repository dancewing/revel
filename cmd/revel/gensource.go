// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"

	"github.com/dancewing/revel"
	"github.com/dancewing/revel/cmd/harness"
)

var cmdGensource = &Command{
	UsageLine: "gensource [import path]",
	Short:     "gensource for a Revel application (e.g. for deployment)",
	Long: `
Gensource for the Revel web application named by the given import path.
Generate source for main.go and routes.go

For example:

    revel gensource github.com/dancewing/examples/chat
`,
}

func init() {
	cmdGensource.Run = genSource
}

func genSource(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "%s\n%s", cmdGensource.UsageLine, cmdGensource.Long)
		return
	}

	appImportPath, mode := args[0], "dev"

	if !revel.Initialized {
		revel.Init(mode, appImportPath, "")
	}

	_, reverr := harness.GenerateSource()
	panicOnError(reverr, "Failed to generate sources and build")

}
