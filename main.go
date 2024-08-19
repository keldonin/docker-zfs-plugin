package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	zfsdriver "github.com/TrilliumIT/docker-zfs-plugin/zfs"
	"github.com/coreos/go-systemd/activation"
	"github.com/docker/go-plugins-helpers/volume"
	log "github.com/sirupsen/logrus"
)

const shutdownTimeout = 10 * time.Second

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
	errCh := make(chan error)

	listeners, _ := activation.Listeners() // wtf coreos, this funciton never returns errors
	if len(listeners) > 1 {
		log.Warn("driver does not support multiple sockets")
	}
	if len(listeners) == 0 {
		log.Debug("launching volume handler.")
		go func() { errCh <- h.ServeUnix("zfs-v2", 0) }()
	} else {
		l := listeners[0]
		log.WithField("listener", l.Addr().String()).Debug("launching volume handler")
		go func() { errCh <- h.Serve(l) }()
	}

	c := make(chan os.Signal)
	defer close(c)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)

	select {
	case err = <-errCh:
		log.WithError(err).Error("error running handler")
		close(errCh)
	case <-c:
	}

	toCtx, toCtxCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer toCtxCancel()
	if sErr := h.Shutdown(toCtx); sErr != nil {
		err = sErr
		log.WithError(err).Error("error shutting down handler")
	}

	if hErr := <-errCh; hErr != nil && !errors.Is(hErr, http.ErrServerClosed) {
		err = hErr
		log.WithError(err).Error("error in handler after shutdown")
	}
}
