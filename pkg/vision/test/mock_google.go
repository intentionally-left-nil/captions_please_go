package vision_test

import (
	"context"
	"testing"

	"github.com/AnilRedshift/captions_please_go/pkg/structured_error"
	"github.com/AnilRedshift/captions_please_go/pkg/vision"
	"github.com/stretchr/testify/assert"
	"golang.org/x/text/language"
)

const DummyGoogleCert = `-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAzIVTdjTgQXBdUfnxVavmr8DPJOZiCRMzrSYYJOaYa/qmShwz
FzlNRVRmCsbBv9+ijphTekif5mimxAnxq1is9qOcV/r1TVLoa7aK/yS2814t16re
l5t21JdIYDeMFoIgiH4ynF4C/Z5pRqX41QRofB3qN63ls/+1lRgifVqUdn6A7iIv
p3Y77ZfUPCTcyDWds2F+zLX7DmSpQhlHvIdrZhBZED52PsV2NaCEOQU46e0IaRV6
6/tZLiJ0AMA/fqVaHaXs2LVzq8zdZbLuxGwfe1Bh6FYYPFRwXx5qTPxEv5WFtYU9
xm0Mk9Lnn/AGfOdPkUMJ2V9wXUir9r7vN31OowIDAQABAoIBAQCUOtF56+rZIuJQ
BtIWIKfqm9jGSr+lCii7BtAa9pJkOF8LeZLB80MAy6HFj7ZfJWvA48Ak8bwKl7C+
huKEKJn7jCtFTNs7NqrDXqMxNt/uVUTueaYoxYGDpT3MlpXOvnNr2eM+l5idTpHI
pYRKh45e3qOhxUSlh+CIddyRc/QESHFl9lQ9fZdOKUlHUhDd1iq8iwlC4gTUZH7Q
oOC9k2lfaBL2Eitk2dvQ4c+AJRVyWdNS//ZGHzN6D0p0hx+neqAVAon1aEVD//yS
o9+vnZENQ3faQSf7Fy7mDTXT9TPVQAbItLsVpHfl470yqTJicWMAjZIlnXyaojJO
9EkSYruBAoGBAPGrkECkKNKYOmkPUOmMWrAVRuKCuOEfaqK6QBpZX7nfsRUOYqbe
JL724FlDHJJz5YLMbHl/66O8w3OgdNz+eVpkBNpJlhkQjPF2lApR/rohmdvoOe5j
gs/oCTSdrdEGk1prcCmZ+IwD8GM56JyUYDyS3ZK4NJ1kJppc056yS+bjAoGBANil
2mVC7AcyH+ZH4b23OmFNHwlSIB0VjbcaiHM3vxp4zzzcIX1SJQVsJ5CiqHfEypVn
hc9hHNAhdJ8o5a1IiJ2vYrh0zGpTj7o/3XSkizJcR/Bzn/80BUO26UC86NwFAJ3g
+4ypR8QFhnqOFQ3z/3bieaamByj4Afq+QgsrOcVBAoGBAJmjOFHgCxPXM0sXMZlI
YV8QJ8BY2rBECMbrIVWe+/xu+WUpgA4Vq8a7rGUTBVcV1xMQYuXbLTMrDha0K5dT
MFMGww8DOSk2HGRlvjfRaN9r/SSQvkOPf9os6a1JkPcR9xvEscnA2QIqfuiWKAtj
SMs5kyNzd/+Xa/M2kFKThy2BAoGADF3jSpZ4XKzKz11ZEHhOF9HMLL8IYECjt0kH
cvRCr2MoCURTkRDIVjfnRkVSsouEOOUQ6VaUy3itbIxsF+klC0NAsmDQbl1Yvfv5
Szg9TeGgpaQkBPBWQJhHVk+yRyTt9RUrpsre8tyR4ZsMrqA3+/RPl2iwzfDiRArq
QDL2eEECgYEA3toHy0CDlcU+AfZxH0RY051385e/qrQPkH5rQ1vSriw7GjlJqKEd
qwH8t5U/BY0rnp2+t1tk91uhqtG0jVRf8C742rZDiXlGXf+pq7ZdiXr1JQBHUDoZ
vf2FL8Z20ZOKoy5CJ26qQiT87BwuL+GS7w+HzjYmCL39SE3QzxY+rs4=
-----END RSA PRIVATE KEY-----`

type MockGoogle struct {
	T             *testing.T
	GetOCRMock    func(url string) (result *vision.OCRResult, err error)
	TranslateMock func(message string) (language.Tag, string, error)
}

func (g *MockGoogle) GetOCR(ctx context.Context, url string) (*vision.OCRResult, structured_error.StructuredError) {
	assert.NotNil(g.T, g.GetOCRMock)
	result, err := g.GetOCRMock(url)
	return result, structured_error.Wrap(err, structured_error.OCRError)
}

func (g *MockGoogle) Translate(ctx context.Context, message string) (language.Tag, string, structured_error.StructuredError) {
	assert.NotNil(g.T, g.TranslateMock)
	tag, result, err := g.TranslateMock(message)
	return tag, result, structured_error.Wrap(err, structured_error.TranslateError)
}

func (g *MockGoogle) Close() error {
	return nil
}
