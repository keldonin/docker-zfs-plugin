package main

import (
	"os"
	"strconv"
	zfsdriver "github.com/keldonin/docker-zfs-plugin/zfs"
	"github.com/coreos/go-systemd/activation"
	"github.com/docker/go-plugins-helpers/volume"
	log "github.com/sirupsen/logrus"
)

func main() {
	debug := os.Getenv("DEBUG")
	if ok, _ := strconv.ParseBool(debug); ok {
		log.SetLevel(log.DebugLevel)
	}

	d, err := zfsdriver.NewZfsDriver()
	if err != nil {
		panic(err)
	}

	h := volume.NewHandler(d)

	listeners, _ := activation.Listeners() // wtf coreos, this funciton never returns errors
	if len(listeners) > 1 {
		log.Warn("driver does not support multiple sockets")
	}
	if len(listeners) == 0 {
		log.Debug("launching volume handler.")
		h.ServeUnix("zfs-v2", 0)
	} else {
		l := listeners[0]
		log.WithField("listener", l.Addr().String()).Debug("launching volume handler")
		h.Serve(l)
	}
}
