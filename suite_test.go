package ceph_e2e

import (
	"encoding/json"
	"github.com/onmetal/cephlet/ori/volume/cmd/volume/app"
	v1alpha12 "github.com/onmetal/onmetal-api/ori/apis/meta/v1alpha1"
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
	RegisterFailHandler(Fail)
	RunSpecs(t, "GRPC Server Suite")
}

var (
	volumeClient v1alpha1.VolumeRuntimeClient
)

var _ = BeforeSuite(func(ctx SpecContext) {
	keyEncryptionKeyFile, err := os.CreateTemp(GinkgoT().TempDir(), "keyencryption")
	Expect(err).NotTo(HaveOccurred())
	defer func() {
		_ = keyEncryptionKeyFile.Close()
	}()
	Expect(os.WriteFile(keyEncryptionKeyFile.Name(), []byte("foofoofoofoofoofoofoofoofoofoofoofoofoofoofoofoofoofoo"), 0666)).To(Succeed())

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
	Expect(app.Run(ctx, opts)).To(Succeed())

	address, err := volume.GetAddressWithTimeout(3*time.Second, opts.Address)
	Expect(err).NotTo(HaveOccurred())

	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	Expect(err).NotTo(HaveOccurred())

	volumeClient = v1alpha1.NewVolumeRuntimeClient(conn)
	DeferCleanup(conn.Close)

	resp, err := volumeClient.ListVolumeClasses(ctx, &v1alpha1.ListVolumeClassesRequest{})
	Expect(resp.VolumeClasses).To(Equal(volumeClasses))

	createResp, err := volumeClient.CreateVolume(ctx, &v1alpha1.CreateVolumeRequest{
		Volume: &v1alpha1.Volume{
			Metadata: &v1alpha12.ObjectMetadata{
				Id: "foo",
			},
			Spec: &v1alpha1.VolumeSpec{
				Class: "foo",
				Resources: &v1alpha1.VolumeResources{
					StorageBytes: 1024 * 1024 * 1024,
				},
			},
		},
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(createResp).To(Equal(v1alpha1.CreateVolumeResponse{
		Volume: &v1alpha1.Volume{
			Metadata: &v1alpha12.ObjectMetadata{
				Id: "foo",
			},
			Spec: &v1alpha1.VolumeSpec{
				Class: "foo",
				Resources: &v1alpha1.VolumeResources{
					StorageBytes: 1024 * 1024 * 1024,
				},
			},
			Status: &v1alpha1.VolumeStatus{
				State: v1alpha1.VolumeState_VOLUME_AVAILABLE,
				Access: &v1alpha1.VolumeAccess{
					Driver:     "",
					Handle:     "",
					Attributes: nil,
					SecretData: nil,
				},
			},
		},
	}))
})
