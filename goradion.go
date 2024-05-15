package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/agejevasv/goradion/internal/radio"
)

func main() {
	cfg := flag.String("s", "", "A link or a path to a stations.csv file")
	ver := flag.Bool("v", false, "Show the version number and quit")
	dbg := flag.Bool("d", false, "Enable debug log (file)")
	flag.Parse()

	radio.InitLog(*dbg)

	if *ver {
		fmt.Println(radio.VersionString())
		os.Exit(0)
	}

	stations, urls := radio.Stations(*cfg)

	player := radio.NewPlayer()
	go player.Start()
	defer player.Quit()

	if err := radio.NewApp(player, stations, urls).Run(); err != nil {
		panic(err)
	}
}
