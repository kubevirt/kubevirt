package virtctl

import (
	"io/ioutil"
	"os"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("virctl command", func() {

	var ctrl *gomock.Controller

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})

	Context("with valid file path", func() {
		tempConfig, err := ioutil.TempFile(os.TempDir(), "kubevirt-test.*.vvv")
		if err != nil {
			Fail("Cannot create the temporary file")
		}
		defer func() {
			err := tempConfig.Close()
			if err != nil {
				Fail("Cannot close the file")
			}
		}()

		data := []byte("vnc default test_vm_123")
		_, err = tempConfig.Write(data)

		osArgs := []string{tempConfig.Name()}

		It("should mark valid file", func() {
			filePath, err := checkForFile(tempConfig.Name())
			Expect(err).To(BeNil(), "should be nil for existing file")
			Expect(filePath).To(Equal(tempConfig.Name()), "marked file path should match")
		})

		It("should parse file properly", func() {
			parsedArgs, err := parseMime(osArgs)

			Expect(err).To(BeNil(), "should parse properly")
			Expect(len(parsedArgs)).To(Equal(3), "parsed arguments should have len 3")
			Expect(parsedArgs[0]).To(Equal("vnc"), "first argument should be")
			Expect(parsedArgs[1]).To(Equal("--namespace=default"), "second argument should be")
			Expect(parsedArgs[2]).To(Equal("test_vm_123"), "third argument should be")
		})
	})

	Context("with invalid file path", func() {
		validFilePath := "./does_not_exist.imaginary"

		It("should mark invalid file", func() {
			filePath, err := checkForFile(validFilePath)
			Expect(err).To(BeNil(), "should do nothing")
			Expect(filePath).To(Equal(""), "marked file path should match")
		})
	})

	Context("with valid file read", func() {
		simulatedFile := `vnc default my-vm`

		It("should parse file properly", func() {
			parsedArgs, err := parseMimeConfig(simulatedFile)
			Expect(err).To(BeNil(), "should parse valid file properly")
			Expect(len(parsedArgs)).To(Equal(3), "parsed arguments should have len 3")
			Expect(parsedArgs[0]).To(Equal("vnc"), "first argument should be")
			Expect(parsedArgs[1]).To(Equal("--namespace=default"), "second argument should be")
			Expect(parsedArgs[2]).To(Equal("my-vm"), "third argument should be")
		})
	})

	Context("with invalid file read", func() {
		It("should return error when without vnc or console", func() {
			simulatedArgs := "systemctl restart cups"

			_, err := parseMimeConfig(simulatedArgs)
			Expect(err.Error()).To(Equal("Protocol have to be one of: vnc, console. Got: systemctl"), "should catch wrong protocol as first argument")
		})

		It("should return error when more than 3 options are passed", func() {
			simulatedArgs := "systemctl restart cups; systemctl disable cups"

			_, err := parseMimeConfig(simulatedArgs)
			Expect(err.Error()).To(Equal("Invalid file format, 3 parameters required, 6 received"), "should catch wrong number of parameters in file")
		})

		DescribeTable("should allow valid name in third token",
			func(token string) {
				simulatedArgs := `vnc cups ` + token

				_, err := parseMimeConfig(simulatedArgs)
				Expect(err).To(BeNil())
			},
			Entry("VM name: vmname", "vmname"),
			Entry("VM name: vm_name", "vm_name"),
			Entry("VM name: vm-name", "vm-name"),
			Entry("VM name: vm-name123", "vm-name123"),
			Entry("VM name: vm-name-123", "vm-name-123"),
			Entry("VM name: vm-name_123", "vm-name_123"),
		)

		DescribeTable("should catch illegal character in second token",
			func(token string) {
				simulatedArgs := `vnc ` + token + `restart cups`

				_, err := parseMimeConfig(simulatedArgs)

				Expect(err).ToNot(BeNil(), "parsing should fail")
				Expect(err.Error()).To(Equal(`Token containing illegal character: ` + token + `restart`))
			},
			Entry("Char ;", ";"),
			Entry("Char [", "["),
			Entry("Char ]", "]"),
			Entry("Char !", "!"),
			Entry("Char @", "@"),
			Entry("Char #", "#"),
			Entry("Char $", "$"),
			Entry("Char %", "%"),
			Entry("Char ^", "^"),
			Entry("Char *", "*"),
			Entry("Char (", "("),
			Entry("Char )", ")"),
			Entry("Char {", "{"),
			Entry("Char }", "}"),
			Entry("Char :", ":"),
			Entry("Char '", "'"),
			Entry(`Char "`, `"`),
			Entry(`Char \`, `\`),
			Entry(`Char |`, `|`),
			Entry(`Char /`, `/`),
			Entry(`Char .`, `.`),
			Entry(`Char ,`, `,`),
			Entry(`Char ~`, `~`),
			Entry("Char `", "`"),
			Entry("Char =", "="),
			Entry("Char +", "+"),
		)
	})

	Context("with invalid temporary config file", func() {
		It("should modify nothing", func() {
			passedArgs := []string{"dummyArgumentThatNeverWillWork.OnThisOrOtherCOmputer"}

			parsedArgs, err := parseMime(passedArgs)

			Expect(err).To(BeNil(), "nothing should happen")
			Expect(parsedArgs).To(BeNil(), "arguments should be put through")
		})

		It("shoudl detect valid file, but mark missing", func() {
			passedArgs := []string{"test.vvv"}

			parsedArgs, err := parseMime(passedArgs)

			Expect(err.Error()).To(Equal("File does not exist"), "nothing should happen")
			Expect(parsedArgs).To(BeNil(), "nothing should be parsed")
		})
	})

	AfterEach(func() {
		ctrl.Finish()
	})
})
