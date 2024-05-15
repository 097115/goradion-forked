package radio

import (
	"bufio"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
)

const defaultStationsURL = "https://gist.githubusercontent.com/agejevasv/" +
	"58afa748a7bc14dcccab1ca237d14a0b/raw/stations.csv"

const defaultStationsCSV = `Jazz Groove,https://audio-edge-cmc51.fra.h.radiomast.io/f0ac4bf3-bbe5-4edb-b828-193e0fdc4f2f
	KCSM Jazz Tonight,https://ice7.securenetsystems.net/KCSM2
	Jazz24,https://prod-52-201-196-36.amperwave.net/ppm-jazz24aac256-ibc1
	Seeburg 1000,https://psn3.prostreaming.net/proxy/seeburg/stream/;
	Chillsky,https://lfhh.radioca.st/stream
	9128live,https://streams.radio.co/s0aa1e6f4a/listen
	Nightride,https://stream.nightride.fm/nightride.ogg
	Jungletrain.net,http://stream1.jungletrain.net:8000
	Prysm Deepinside,https://n16a-eu.rcs.revma.com/7tkkn1yuhnruv
	Deep Motion FM,https://vm.motionfm.com/motionone_aacp
	Lounge Motion FM,https://vm.motionfm.com/motionthree_aacp
	Smooth Motion FM,https://vm.motionfm.com/motiontwo_aacp
	SomaFM: Beat Blender,https://somafm.com/beatblender.pls
	SomaFM: Bossa Beyond,https://somafm.com/bossa256.pls
	SomaFM: Groove Salad Classic,https://somafm.com/gsclassic130.pls
	SomaFM: Sonic Universe,https://somafm.com/sonicuniverse256.pls
	SomaFM: DEF CON Radio,http://somafm.com/defcon.pls
	SomaFM: Fluid,http://somafm.com/fluid130.pls
	SomaFM: Illinois Street Lounge,http://somafm.com/illstreet.pls
	SomaFM: Vaporwaves,http://somafm.com/vaporwaves.pls
	SomaFM: Drone Zone,http://somafm.com/dronezone.pls
	SomaFM: Deep Space One,https://somafm.com/deepspaceone130.pls
	Big FM,https://stream.bigfm.de/oldschoolrap/aac-128/radiode`

func Stations(sta string) ([]string, []string) {
	var scanner *bufio.Scanner

	if sta == "" {
		go cacheDefaultStations()
		if s, err := cachedDefaultStations(); err != nil {
			scanner = bufio.NewScanner(strings.NewReader(defaultStationsCSV))
		} else {
			scanner = bufio.NewScanner(strings.NewReader(string(s)))
		}
	} else if strings.HasPrefix(sta, "http") {
		s, err := fetchStations(sta)
		if err != nil {
			s = defaultStationsCSV
		}
		scanner = bufio.NewScanner(strings.NewReader(s))
	} else {
		file, err := os.Open(sta)
		if err != nil {
			panic(err)
		}
		defer file.Close()
		scanner = bufio.NewScanner(file)
	}

	stat := make([]string, 0)
	urls := make([]string, 0)

	for scanner.Scan() {
		d := strings.Split(scanner.Text(), ",")
		stat = append(stat, strings.Trim(d[0], " "))
		urls = append(urls, strings.Trim(d[1], " "))
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}

	return stat, urls
}

func cacheDefaultStations() {
	s, err := fetchStations(defaultStationsURL)
	if err != nil {
		log.Panicln(err)
		return
	}
	dir, _ := os.UserHomeDir()
	ioutil.WriteFile(path.Join(dir, "goradion.csv"), []byte(s), 0644)
}

func cachedDefaultStations() ([]byte, error) {
	dir, _ := os.UserHomeDir()
	return ioutil.ReadFile(path.Join(dir, "goradion.csv"))
}

func fetchStations(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}