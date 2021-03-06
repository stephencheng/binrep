package command

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/pkg/errors"

	"github.com/yuuki/binrep/pkg/release"
	"github.com/yuuki/binrep/pkg/storage"
)

// PushParam represents the option parameter of `push`.
type PushParam struct {
	Timestamp    string
	KeepReleases int
	Force        bool
}

// Push pushes the binary files of binPaths as release of the name(<host>/<user>/<project>).
func Push(param *PushParam, name string, binPaths []string) error {
	// TODO: Validate the same file name
	bins := make([]*release.Binary, 0, len(binPaths))
	for _, binPath := range binPaths {
		file, err := os.Open(binPath)
		if err != nil {
			return errors.Wrapf(err, "failed to open %v", binPath)
		}
		fi, err := file.Stat()
		if err != nil {
			return errors.Wrapf(err, "failed to stat %q", file.Name())
		}
		bin, err := release.BuildBinary(filepath.Base(file.Name()), fi.Mode(), file)
		if err != nil {
			return err
		}
		bins = append(bins, bin)
	}

	sess := session.New()
	st := storage.New(sess)

	if !param.Force {
		ok, err := st.ExistRelease(name)
		if err != nil {
			return err
		}
		if ok {
			ok, err := st.HaveSameChecksums(name, bins)
			if err != nil {
				return err
			}
			if ok {
				log.Println("Skip pushing the binaries because they have the same checksum with the latest binaries on the remote storage")
				return nil
			}
		}
	}

	log.Println("-->", "Uploading", binPaths)

	rel, err := st.CreateRelease(name, release.Now(), bins)
	if err != nil {
		return err
	}

	log.Println("Uploaded", "to", rel.URL)

	log.Println("--> Cleaning up the old releases")

	timestamps, err := st.PruneReleases(name, param.KeepReleases)
	if err != nil {
		return err
	}

	log.Println("Cleaned", "up", strings.Join(timestamps, ","))

	return nil
}
