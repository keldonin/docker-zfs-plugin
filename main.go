package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/coreos/go-systemd/activation"
	"github.com/docker/go-plugins-helpers/volume"
	zfsdriver "github.com/keldonin/docker-zfs-plugin/zfs"
	log "github.com/sirupsen/logrus"
)

// SimpleFormatter génère des logs simples sans encapsulation
type SimpleFormatter struct{}

func (f *SimpleFormatter) Format(entry *log.Entry) ([]byte, error) {
	message := "zfs:" + log.GetLevel().String() + ":" + entry.Message

	// Ajouter les champs importants seulement
	if dataset, exists := entry.Data["dataset"]; exists {
		message += fmt.Sprintf(" (dataset: %v)", dataset)
	}
	if err, exists := entry.Data["error"]; exists {
		message += fmt.Sprintf(" (error: %v)", err)
	}

	return []byte(message + "\n"), nil
}

func main() {

	// Configuration du logger - alignement avec Docker daemon
	log.SetOutput(os.Stdout)
	log.SetFormatter(&SimpleFormatter{})

	// Par défaut, alignement avec le niveau de Docker (généralement info)
	log.SetLevel(log.InfoLevel)

	// Permettre de forcer un niveau via une variable d'environnement spécifique au plugin
	loglevel := strings.ToLower(os.Getenv("LOGLEVEL"))
	switch loglevel {
	case "panic":
		log.Info("zfs volume plugin log level set to panic")
		log.SetLevel(log.PanicLevel)
	case "fatal":
		log.Info("zfs volume plugin log level set to fatal")
		log.SetLevel(log.FatalLevel)
	case "error":
		log.Info("zfs volume plugin log level set to error")
		log.SetLevel(log.ErrorLevel)
	case "warn", "warning":
		log.Info("zfs volume plugin log level set to warn")
		log.SetLevel(log.WarnLevel)
	case "debug":
		log.Info("zfs volume plugin log level set to debug")
		log.SetLevel(log.DebugLevel)
	case "trace":
		log.Info("zfs volume plugin log level set to trace")
		log.SetLevel(log.TraceLevel)
	default:
		// we "info". We are already at the right level
		//log.SetLevel(log.InfoLevel)
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
