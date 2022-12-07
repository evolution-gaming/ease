// Copyright ©2022 Evolution. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import "flag"

type globalFlags struct {
	ConfFile string
	Debug    bool
}

func (g *globalFlags) Register(fs *flag.FlagSet) {
	fs.BoolVar(&g.Debug, "debug", false, "Enable debug logging (optional)")
	fs.StringVar(&g.ConfFile, "conf", "", "Application configuration file path (optional)")
}
