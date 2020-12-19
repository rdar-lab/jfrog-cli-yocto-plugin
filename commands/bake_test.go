package commands

import (
	"github.com/jfrog/jfrog-client-go/artifactory/buildinfo"
	"github.com/stretchr/testify/assert"
	"testing"
)

const manifestFileContent = "base-files qemux86 3.0.14\nbase-passwd i586 3.5.29\nbusybox i586 1.24.1\nbusybox-hwclock i586 1.24.1\nbusybox-syslog i586 1.24.1\nbusybox-udhcpc i586 1.24.1\neudev i586 3.2\ninit-ifupdown qemux86 1.0\ninitscripts i586 1.0\ninitscripts-functions i586 1.0\nkernel-4.8.26-yocto-standard qemux86 4.8.26+git0+1c60e003c7_27efc3ba68\nkernel-module-uvesafb qemux86 4.8.26+git0+1c60e003c7_27efc3ba68\nlibblkid1 i586 2.28.1\nlibc6 i586 2.24\nlibkmod2 i586 23+git0+65a885df5f\nlibuuid1 i586 2.28.1\nlibz1 i586 1.2.8\nmodutils-initscripts i586 1.0\nnetbase i586 5.3\npackagegroup-core-boot qemux86 1.0\nrun-postinsts all 1.0\nsysvinit i586 2.88dsf\nsysvinit-inittab qemux86 2.88dsf\nsysvinit-pidof i586 2.88dsf\nudev-cache i586 3.2\nupdate-alternatives-opkg i586 0.3.2+git0+3ffece9bf1\nupdate-rc.d all 0.7\nv86d qemux86 0.1.10\n"

func TestManifestParsing(t *testing.T) {
	resultList := parseManifestContent(manifestFileContent, make([]buildinfo.Dependency, 0))
	assert.NotNil(t, resultList)
	assert.NotEmpty(t, resultList)
	assert.Equal(t, 28, len(resultList))
	assert.True(t, contains(resultList,
		buildinfo.Dependency{
			Id:     "base-passwd:3.5.29",
			Type:   "os-package",
			Scopes: []string{"i586"},
		}))
}
