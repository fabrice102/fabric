package decorator

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/config"
	"github.com/hyperledger/fabric/core/handlers/decoration"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/spf13/viper"
)

var decoratorLogger = flogging.MustGetLogger("decorator")

// NewFDecorator creates a new decorator
func NewFDecorator() decoration.Decorator {
	// Load decoration.yaml

	decoratorLogger.Infof("New FDecorator...")

	v := &viper.Viper{}
	err := config.InitViper(v, "decoration")
	if err != nil { // Handle errors reading the config file
		decoratorLogger.Infof("fatal error when initializing %s config : %s\n", "decoration", err)
	}

	v.SetConfigType("yaml")
	err = v.ReadInConfig() // Find and read the config file
	if err != nil {        // Handle errors reading the config file
		decoratorLogger.Infof("fatal error when reading %s config : %s\n", "decoration", err)
	}

	decoratorLogger.Infof("loaded decoration.yaml")
	keys := v.AllKeys()
	values := make(map[string][]byte)
	decoratorLogger.Infof("# keys %v", len(keys))
	for _, k := range keys {
		decoratorLogger.Infof("adding key %v:%v", k, v.GetString(k))
		values[k] = []byte(v.GetString(k))
	}

	return &fDecorator{values: values}
}

type fDecorator struct {
	values map[string][]byte
}

// Decorate decorates a chaincode input by changing it
func (d *fDecorator) Decorate(proposal *peer.Proposal, input *peer.ChaincodeInput) *peer.ChaincodeInput {
	decoratorLogger.Infof("Decorate %d", len(d.values))
	for k, v := range d.values {
		decoratorLogger.Infof("adding key %v:%v", k, v)
		input.Decorations[k] = v
	}

	return input
}
