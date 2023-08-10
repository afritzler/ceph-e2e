package ceph_e2e

import (
	"context"
	"encoding/json"
	"github.com/onmetal/cephlet/ori/volume/cmd/volume/app"
	"github.com/onmetal/onmetal-api/ori/apis/volume/v1alpha1"
	"github.com/onmetal/onmetal-api/ori/remote/volume"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGRPCServer(t *testing.T) {
	if os.Getenv("E2E_TESTS") == "true" {
		t.Skip("Skipping test because E2E_TESTS is set to true")
	}

	sConfig, rConfig := GinkgoConfiguration()
	rConfig.FullTrace = true
	RegisterFailHandler(Fail)
	RunSpecs(t, "GRPC Server Suite", sConfig, rConfig)
}

var (
	volumeClient v1alpha1.VolumeRuntimeClient
)

var _ = BeforeEach(func() {
	keyEncryptionKeyFile, err := os.CreateTemp(GinkgoT().TempDir(), "keyencryption")
	Expect(err).NotTo(HaveOccurred())
	defer func() {
		_ = keyEncryptionKeyFile.Close()
	}()
	Expect(os.WriteFile(keyEncryptionKeyFile.Name(), []byte("foooooooooooooooooooooooooooooooooooooooooooooooooooo"), 0666)).To(Succeed())

	volumeClasses := []v1alpha1.VolumeClass{{
		Name: "foo",
		Capabilities: &v1alpha1.VolumeClassCapabilities{
			Tps:  100,
			Iops: 100,
		},
	}}
	volumeClassesData, err := json.Marshal(volumeClasses)
	Expect(err).NotTo(HaveOccurred())

	volumeClassesFile, err := os.CreateTemp(GinkgoT().TempDir(), "volumeclasses")
	Expect(err).NotTo(HaveOccurred())
	defer func() {
		_ = volumeClassesFile.Close()
	}()
	Expect(os.WriteFile(keyEncryptionKeyFile.Name(), volumeClassesData, 0666)).To(Succeed())

	srvCtx, cancel := context.WithCancel(context.Background())
	DeferCleanup(cancel)

	opts := app.Options{
		Address:                    "/var/run/cephlet-volume.sock",
		PathSupportedVolumeClasses: volumeClassesFile.Name(),
		Ceph: app.CephOptions{
			Monitors:             os.Getenv("CEPH_MONITORS"),
			User:                 os.Getenv("CEPH_USERNAME"),
			KeyFile:              os.Getenv("CEPH_KEY"),
			Pool:                 os.Getenv("CEPH_POOLNAME"),
			Client:               os.Getenv("CEPH_CLIENTNAME"),
			KeyEncryptionKeyPath: keyEncryptionKeyFile.Name(),
		},
	}
	go func() {
		defer GinkgoRecover()
		Expect(app.Run(srvCtx, opts)).To(Succeed())
	}()

	address, err := volume.GetAddressWithTimeout(3*time.Second, opts.Address)
	Expect(err).NotTo(HaveOccurred())

	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	Expect(err).NotTo(HaveOccurred())

	volumeClient = v1alpha1.NewVolumeRuntimeClient(conn)
	DeferCleanup(conn.Close)
})
