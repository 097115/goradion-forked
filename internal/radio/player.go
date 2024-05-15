package radio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"
)

const defaultVolume = 80

type Player struct {
	sync.Mutex
	Info    chan Info
	cmd     *exec.Cmd
	info    *Info
	url     string
	volume  int
	stopped chan struct{}
}

type Control struct {
	start chan bool
	stop  chan bool
}

type Info struct {
	Status string
	Song   string
	Volume int
}

func NewPlayer() *Player {
	return &Player{
		Info:   make(chan Info),
		volume: defaultVolume,
		info: &Info{
			Volume: defaultVolume,
		},
		stopped: make(chan struct{}),
	}
}

func (p *Player) Start() {
	p.Lock()
	p.cmd = exec.Command(
		"mpv",
		"-no-video",
		"--idle",
		fmt.Sprintf("--volume=%d", defaultVolume),
		fmt.Sprintf("--input-ipc-server=%s", socket),
	)

	if err := p.cmd.Start(); err != nil {
		fmt.Println("'mpv' was not found in $PATH")
		fmt.Println("Please install 'mpv' using your package manager or visit https://mpv.io for more info.")
		os.Exit(1)
	}

	for i := 1; !p.mvpIsListening() && i <= 10; i++ {
		if i == 10 {
			fmt.Println("mpv failed to start, quitting")
			os.Exit(1)
		}
		log.Printf("waiting for mpv +%d ms\n", 8<<i)
		time.Sleep((8 << i) * time.Millisecond)
	}

	p.Unlock()
	log.Println("mpv is ready")
}

func (p *Player) SetSongTitle(artist, title string) {
	if artist != "" {
		p.info.Song = fmt.Sprintf("%s - %s", artist, title)
		artist, title = "", ""
	} else {
		p.info.Song = title
	}
	p.Info <- *p.info
}

func (p *Player) Toggle(station, url string) {
	p.Lock()
	defer p.Unlock()

	if p.url != "" {
		p.Stop()

		if url == p.url {
			p.url = ""
			return
		}
	}

	p.info.Status = station
	p.info.Song = ""
	p.info.Volume = p.volume
	p.Info <- *p.info
	p.Load(url)
}

func (p *Player) VolumeUp() {
	p.Lock()
	defer p.Unlock()

	defer func() {
		p.info.Volume = p.volume
		p.Info <- *p.info
	}()

	if p.volume == 100 {
		return
	}

	log.Printf("setting volume %d\n", p.volume+5)
	cmd := fmt.Sprintf(`{"command": ["set_property", "volume", %d]}%s`, p.volume+5, "\n")
	p.writeToMPV([]byte(cmd))
	p.volume += 5
}

func (p *Player) VolumeDn() {
	p.Lock()
	defer p.Unlock()

	defer func() {
		p.info.Volume = p.volume
		p.Info <- *p.info
	}()

	if p.volume == 0 {
		return
	}

	log.Printf("setting volume %d\n", p.volume-5)
	cmd := fmt.Sprintf(`{"command": ["set_property", "volume", %d]}%s`, p.volume-5, "\n")
	p.writeToMPV([]byte(cmd))
	p.volume -= 5
}

func (p *Player) Stop() {
	log.Printf("stopping %s\n", p.url)
	cmd := fmt.Sprintf(`{"command": ["stop"]}%s`, "\n")
	p.writeToMPV([]byte(cmd))
	p.info.Status = "Stopped"
	p.info.Song = ""
	p.Info <- *p.info
	p.stopped <- struct{}{}
}

func (p *Player) Load(url string) {
	p.SetSongTitle("", "Loading...")
	log.Printf("loading %s\n", url)
	cmd := fmt.Sprintf(`{"command": ["loadfile", "%s"]}%s`, url, "\n")
	p.writeToMPV([]byte(cmd))
	p.url = url
	go p.readMetadata()
}

func (p *Player) Quit() {
	log.Println("quitting mpv")
	cmd := fmt.Sprintf(`{"command": ["quit", 9]}%s`, "\n")

	if ok := p.writeToMPV([]byte(cmd)); !ok && p.cmd != nil {
		log.Println("mpv failed to quit via socket")
		p.cmd.Process.Signal(os.Kill)
		p.cmd.Wait()
	}
}

func (p *Player) readMetadata() {
	var res map[string]any

	t := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-t.C:
			cmd := fmt.Sprintf(`{"command": ["get_property", "metadata"]}%s`, "\n")

			data, err := p.readFromMPV([]byte(cmd))
			if err != nil {
				log.Println(err)
				continue
			}

			if err = json.Unmarshal(data, &res); err != nil {
				log.Println(err)
				continue
			}

			if res["data"] != nil {
				meta := res["data"].(map[string]any)

				if t, ok := meta["icy-title"]; ok {
					p.SetSongTitle("", t.(string))
				} else {
					a, ok1 := meta["artist"]
					t, ok2 := meta["title"]
					if ok1 && ok2 {
						p.SetSongTitle(a.(string), t.(string))
					}
				}
			}
		case <-p.stopped:
			return
		}

	}
}

func (p *Player) writeToMPV(data []byte) bool {
	c, err := netDial()

	if err != nil {
		log.Println(err, string(data))
		return false
	}

	defer c.Close()

	if _, err = c.Write(data); err != nil {
		log.Println(err, string(data))
		return false
	}

	return true
}

func (p *Player) mvpIsListening() bool {
	_, err := netDial()
	return err == nil
}

func (p *Player) readFromMPV(data []byte) ([]byte, error) {
	c, err := netDial()

	if err != nil {
		return nil, err
	}

	defer c.Close()

	if _, err = c.Write(data); err != nil {
		return nil, err
	}

	res := make([]byte, 1024)
	if _, err = c.Read(res); err != nil {
		return nil, err
	}

	return bytes.Trim(res, "\x00"), nil
}
