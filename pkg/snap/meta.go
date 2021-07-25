package snap

import (
	"errors"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
)

type SnapMeta struct {
	Name          string   `yaml:"name"`
	Version       string   `yaml:"version"`
	Summary       string   `yaml:"summary"`
	Description   string   `yaml:"description"`
	Type          string   `yaml:"type"`
	Architectures []string `yaml:"architectures"`
	Confinement   string   `yaml:"confinement"`
	Grade         string   `yaml:"grade"`
	Base          string   `yaml:"base"`
}

// GetSnapMetaFromFile will return SnapMeta from a byte array representing a snap file
// This is an inefficient but expedient process
func GetSnapMetaFromFile(snapFilePath string, workingDirectory string) (*SnapMeta, error) {
	bytes, err := ioutil.ReadFile(snapFilePath)
	if err == nil {
		return GetSnapMetaFromBytes(bytes, workingDirectory)
	}

	return nil, err
}

func GetSnapMetaFromBytes(bytes []byte, workingDirectory string) (*SnapMeta, error) {
	err := errors.New("there was an error getting snap meta")
	tmpFilePath := path.Join(workingDirectory, uuid.New().String()+".snap")
	err = ioutil.WriteFile(tmpFilePath, bytes, 0755)
	if err == nil {
		defer func(name string) {
			errIn := os.Remove(name)
			if errIn != nil {
				logrus.Error(errIn)
			}
		}(tmpFilePath)
		err = os.Chdir(workingDirectory)
		if err == nil {
			cmd := exec.Command("unsquashfs", tmpFilePath, "-e", "meta/snap.yaml")
			defer func() {
				errIn := os.RemoveAll(path.Join(workingDirectory, "squashfs-root"))
				if errIn != nil {
					logrus.Error(err)
				}
			}()
			cmd.Stderr = os.Stderr
			err = cmd.Run()
			if err == nil {
				bytes, err = ioutil.ReadFile(path.Join(workingDirectory, "squashfs-root", "meta", "snap.yaml"))
				if err == nil {
					var snapMeta SnapMeta
					err = yaml.Unmarshal(bytes, &snapMeta)
					if err == nil {
						return &snapMeta, nil
					}
				}
			}
		}
	}

	return nil, err
}
