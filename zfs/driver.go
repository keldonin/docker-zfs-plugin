package zfsdriver

import (
	"errors"
	"time"

	"github.com/clinta/go-zfs"
	"github.com/docker/go-plugins-helpers/volume"
	log "github.com/sirupsen/logrus"
)

//ZfsDriver implements the plugin helpers volume.Driver interface for zfs
type ZfsDriver struct {
	volume.Driver

	//The volumes the plugin is tracking, use empty struct as value to mimic a set
	volumes   map[string]struct{}
}

//NewZfsDriver returns the plugin driver object
func NewZfsDriver(rootDatasets ...string) (*ZfsDriver, error) {
	log.Debug("Creating new ZfsDriver.")

	if len(dss) < 1 {
		return nil, fmt.Errorf("No datasets specified")
	}

	zd := &ZfsDriver{
		volumes: map[string]struct{}{},
	}

	//Load any datasets under a tracked root dataset
	err := zd.loadDatasetState(rootDatasets)
	if err != nil {
		return nil, err
	}

	return zd, nil
}

func (zd *ZfsDriver) loadDatasetState(rootDatasets []string) (error) {
	for _, rdsn := range rootDatasets {
		rds, err := zfs.GetDataset(rdsn)
		if err != nil {
			log.WithField("RootDatasetName", rdsn).Error("Failed to get root dataset")
			continue
		}

		dsl, err := rds.DatasetList()
		if err != nil {
			return err
		}

		for _, ds := range dsl {
			zd.volumes[ds.Name] = struct{}{}
		}
	}

	return nil
}

//Create creates a new zfs dataset for a volume
func (zd *ZfsDriver) Create(req *volume.CreateRequest) error {
	log.WithField("Request", req).Debug("Create")

	if zfs.DatasetExists(req.Name) {
		return errors.New("Volume already exists")
	}

	_, err := zfs.CreateDatasetRecursive(req.Name, req.Options)
	if err != nil {
		return err
	}

	zd.volumes[req.Name] = struct{}{}

	return err
}

//List returns a list of zfs volumes on this host
func (zd *ZfsDriver) List() (*volume.ListResponse, error) {
	log.Debug("List")
	var vols []*volume.Volume

	for dsn, _ := range zd.volumes {
		vol, err := zd.getVolume(dsn)
		if err != nil {
			log.WithField("DatasetName", dsn).Error("Failed to get dataset info")
			continue
		}
		vols = append(vols, vol)
	}

	return &volume.ListResponse{Volumes: vols}, nil
}

//Get returns the volume.Volume{} object for the requested volume
//nolint: dupl
func (zd *ZfsDriver) Get(req *volume.GetRequest) (*volume.GetResponse, error) {
	log.WithField("Request", req).Debug("Get")

	v, err := zd.getVolume(req.Name)
	if err != nil {
		return nil, err
	}

	return &volume.GetResponse{Volume: v}, nil
}

func (zd *ZfsDriver) getVolume(name string) (*volume.Volume, error) {
	ds, err := zfs.GetDataset(name)
	if err != nil {
		return nil, err
	}

	mp, err := ds.GetMountpoint()
	if err != nil {
		return nil, err
	}

	ts, err := ds.GetCreation()
	if err != nil {
		log.WithError(err).Error("Failed to get creation property from zfs dataset")
		return &volume.Volume{Name: name, Mountpoint: mp}, nil
	}

	return &volume.Volume{Name: name, Mountpoint: mp, CreatedAt: ts.Format(time.RFC3339)}, nil
}

func (zd *ZfsDriver) getMP(name string) (string, error) {
	ds, err := zfs.GetDataset(name)
	if err != nil {
		return "", err
	}

	return ds.GetMountpoint()
}

//Remove destroys a zfs dataset for a volume
func (zd *ZfsDriver) Remove(req *volume.RemoveRequest) error {
	log.WithField("Request", req).Debug("Remove")

	ds, err := zfs.GetDataset(req.Name)
	if err != nil {
		return err
	}


	err = ds.Destroy()
	if err != nil {
		return err
	}

	delete(zd.volumes, req.Name)

	return nil
}

//Path returns the mountpoint of a volume
//nolint: dupl
func (zd *ZfsDriver) Path(req *volume.PathRequest) (*volume.PathResponse, error) {
	log.WithField("Request", req).Debug("Path")

	mp, err := zd.getMP(req.Name)
	if err != nil {
		return nil, err
	}

	return &volume.PathResponse{Mountpoint: mp}, nil
}

//Mount returns the mountpoint of the zfs volume
//nolint: dupl
func (zd *ZfsDriver) Mount(req *volume.MountRequest) (*volume.MountResponse, error) {
	log.WithField("Request", req).Debug("Mount")
	mp, err := zd.getMP(req.Name)
	if err != nil {
		return nil, err
	}

	return &volume.MountResponse{Mountpoint: mp}, nil
}

//Unmount does nothing because a zfs dataset need not be unmounted
func (zd *ZfsDriver) Unmount(req *volume.UnmountRequest) error {
	log.WithField("Request", req).Debug("Unmount")
	return nil
}

//Capabilities sets the scope to local as this is a local only driver
func (zd *ZfsDriver) Capabilities() *volume.CapabilitiesResponse {
	log.Debug("Capabilities")
	return &volume.CapabilitiesResponse{Capabilities: volume.Capability{Scope: "local"}}
}
