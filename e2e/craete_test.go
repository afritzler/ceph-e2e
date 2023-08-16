package e2e

import (
	v1alpha12 "github.com/onmetal/onmetal-api/ori/apis/meta/v1alpha1"
	"github.com/onmetal/onmetal-api/ori/apis/volume/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Create Volume", func() {
	It("should get the supported volume classes", func(ctx SpecContext) {
		resp, err := volumeClient.ListVolumeClasses(ctx, &v1alpha1.ListVolumeClassesRequest{})
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.VolumeClasses).To(Equal([]*v1alpha1.VolumeClass{{
			Name: "foo",
			Capabilities: &v1alpha1.VolumeClassCapabilities{
				Tps:  100,
				Iops: 100,
			},
		}}))
	})

	It("should create a volume", func(ctx SpecContext) {
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
		Expect(createResp).Should(SatisfyAll(
			HaveField("Volume.Metadata.Id", Not(BeEmpty())),
			HaveField("Volume.Spec.Image", Equal("")),
			HaveField("Volume.Spec.Class", Equal("foo")),
			HaveField("Volume.Spec.Resources.StorageBytes", Equal(uint64(1024*1024*1024))),
			HaveField("Volume.Spec.Encryption", BeNil()),
			HaveField("Volume.Status.State", Equal(v1alpha1.VolumeState_VOLUME_PENDING)),
			HaveField("Volume.Status.Access", BeNil()),
		))

		resp, err := volumeClient.ListVolumes(ctx, &v1alpha1.ListVolumesRequest{
			Filter: &v1alpha1.VolumeFilter{
				Id: createResp.Volume.Metadata.Id,
			},
		})

		Expect(resp.Volumes).NotTo(BeEmpty())
	})
})
