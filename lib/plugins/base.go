package plugins

import (
	"github.com/sirupsen/logrus"
)

// Plugin base. Plugin implementations should contain this for logging.
type base struct {
	logrus.FieldLogger
}

func newBase(name string) base {
	return base{logrus.New().WithField("module", name)}
}
