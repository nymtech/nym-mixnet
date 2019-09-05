package main

import "github.com/tav/golly/optparse"

func main() {
	var logo = `
  _                      _      
 | |    ___   ___  _ __ (_)_  __
 | |   / _ \ / _ \| '_ \| \ \/ /
 | |___ (_) | (_) | |_) | |>  < 
 |_____\___/ \___/| .__/|_/_/\_\
		  |_|            (client)
		  
		  `
	cmds := map[string]func([]string, string){
		"run": cmdRun,
	}
	info := map[string]string{
		"run": "Run a persistent Loopix client process",
	}
	optparse.Commands("loopix-client", "0.0.1", cmds, info, logo)
}
