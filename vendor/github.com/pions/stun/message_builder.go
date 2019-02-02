package stun

import (
	"github.com/pkg/errors"
)

// Build return messsage which is built from class, method, transactionID, and pack message using attribute
func Build(class MessageClass, method Method, transactionID []byte, attrs ...Attribute) (*Message, error) {

	m := &Message{
		Class:         class,
		Method:        method,
		TransactionID: transactionID,
	}

	m.Raw = make([]byte, messageHeaderLength)

	setMessageType(m.Raw[messageHeaderStart:2], class, method)

	copy(m.Raw[transactionIDStart:], m.TransactionID)

	for _, v := range attrs {
		err := v.Pack(m)
		if err != nil {
			return nil, errors.Wrap(err, "failed packing attribute")
		}
	}

	return m, nil
}
