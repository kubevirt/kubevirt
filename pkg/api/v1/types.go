package v1

type VM struct {
	Name string `xml:"name" valid:"required"`
	UUID string `xml:"uuid" valid:"uuid"`
}
