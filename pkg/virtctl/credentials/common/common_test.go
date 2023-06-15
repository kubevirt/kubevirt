package common_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/testutils"

	"kubevirt.io/kubevirt/pkg/virtctl/credentials/common"
)

var _ = Describe("Credentials common", func() {
	Context("ValidateSshPublicKey", func() {
		It("should accept a valid ssh public key", func() {
			const testKey = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDR2Ah+NcKPU9wDXP7DibuXrkvXCL/YH/w++3M3zZK27WSfjngsawM/Kai8oGXwmjFCprP77COkdBqg2Dpr/ulQ/7h4GwVb/Cjcwov/LOWg5aRAUa1NYRZ75CErMuGW9kSAd42mxeSslLK91hdlCFJP3qMPbkTvlrGAw+6WzwQEmQA1S1D7KC1yJTW6gtgkkKVYNnOhvuGDrCzoOyxb1SfjAhKSk3OkkotdBlWK8TWynGkYhptLAP9pQvCgtRMJPBQ6OWjVV5qkT6yY2hjG6frYnwDotI5OXdOBjbx0Oaa3sFRC983YDIh9lbEKeQxckykg9Iys2fT/NZUbze46hSA/8bG4hDqU0X7+dHN+Ite2/vRjEeaRaWzm9t7+/nxzxibr2x38fkxtNwGYv6VHTyoBTVj/mVqku+NM7pzGGD5X2nB28gbJTCnRPtd4kLIHfg7IYjfHpIBXwfq5jnRlYrIraqkEljZ6iAF4xZGQkQYZQhhwNErJ4+cOFadwG11pdhs= test-key-1"

			Expect(common.ValidateSshPublicKey(testKey)).To(Succeed())
		})

		It("should reject invalid ssh public key", func() {
			const incorrectKey = "this is not a valid key"

			Expect(common.ValidateSshPublicKey(incorrectKey)).ToNot(Succeed())
		})

		It("should reject multiple ssh public keys", func() {
			const testKey1 = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDR2Ah+NcKPU9wDXP7DibuXrkvXCL/YH/w++3M3zZK27WSfjngsawM/Kai8oGXwmjFCprP77COkdBqg2Dpr/ulQ/7h4GwVb/Cjcwov/LOWg5aRAUa1NYRZ75CErMuGW9kSAd42mxeSslLK91hdlCFJP3qMPbkTvlrGAw+6WzwQEmQA1S1D7KC1yJTW6gtgkkKVYNnOhvuGDrCzoOyxb1SfjAhKSk3OkkotdBlWK8TWynGkYhptLAP9pQvCgtRMJPBQ6OWjVV5qkT6yY2hjG6frYnwDotI5OXdOBjbx0Oaa3sFRC983YDIh9lbEKeQxckykg9Iys2fT/NZUbze46hSA/8bG4hDqU0X7+dHN+Ite2/vRjEeaRaWzm9t7+/nxzxibr2x38fkxtNwGYv6VHTyoBTVj/mVqku+NM7pzGGD5X2nB28gbJTCnRPtd4kLIHfg7IYjfHpIBXwfq5jnRlYrIraqkEljZ6iAF4xZGQkQYZQhhwNErJ4+cOFadwG11pdhs= test-key-1"
			const testKey2 = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCx0qmeOuJUi9Sh05Aq/OqxKA4gY+ZuJMEFG9eKQA0nCyb+yBXmVg3T0Leg9JJl5wzygSHyeDstuB6kdGzspufTRQ2YXf0RqlaRfSdc06LlDoc1af4Z6W3Hy4NMwK/nQR9b4Dx8mDLgnxqjueIOu3yZN3ZGr7xsZr+dsygPJQfLSGzbMQ71U/Rh9ETvIU8/aY0hVWb0rMpnQ0X1NBDfqqwSAx9I3kdn1TWkaIDM++lB+g02QsKkTj/MOFBa9gweI0jmjFbbGfwKrTUFLTNYr5M80/Qoj2/KPMEhlIQMBMTNPS9EtgqzlPZyj7Bnmh1UYdMcqYklOhqOJ/rXNlcAIlkA/MMpb/LMCLQUvJuJ51fPaZIqBqxtvY9wVs+CtpjWmouBmjtKe57EadCTyTjuZkxQihTzINyXETjw9U0wnaMJQVhexTjZmR6p7Utz+MoU0R12gfQUirVYX3zQdSQbe/aqX6vbuct+/zoWjkQCdoGABkBP7Y4/FFnBd4hnVJaRRes= test-key-2"

			err := common.ValidateSshPublicKey(testKey1 + "\n" + testKey2)
			Expect(err).To(MatchError(ContainSubstring("only one key can be provided")))
		})
	})
})

func TestCommon(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}
