package srtp

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"testing"
)

type mockKeyingMaterialExporter struct {
	exported []byte
}

func (m *mockKeyingMaterialExporter) ExportKeyingMaterial(label string, context []byte, length int) ([]byte, error) {
	if label != labelExtractorDtlsSrtp {
		return nil, fmt.Errorf("exporter called with wrong label: %s", label)
	}

	m.exported = make([]byte, length)
	if _, err := rand.Read(m.exported); err != nil {
		return nil, fmt.Errorf("failed to create random bytes: %v", err)
	}

	return m.exported, nil
}

func TestExtractSessionKeysFromDTLS(t *testing.T) {
	tt := []struct {
		config *Config
	}{
		{&Config{Profile: ProtectionProfileAes128CmHmacSha1_80}},
	}

	m := &mockKeyingMaterialExporter{}

	for i, tc := range tt {
		// Test client
		err := tc.config.ExtractSessionKeysFromDTLS(m, true)
		if err != nil {
			t.Errorf("failed to extract keys for %d-client: %v", i, err)
		}

		keys := tc.config.Keys
		clientMaterial := append([]byte{}, keys.LocalMasterKey...)
		clientMaterial = append(clientMaterial, keys.RemoteMasterKey...)
		clientMaterial = append(clientMaterial, keys.LocalMasterSalt...)
		clientMaterial = append(clientMaterial, keys.RemoteMasterSalt...)

		if !bytes.Equal(clientMaterial, m.exported) {
			t.Errorf("material reconstruction failed for %d-client:\n%#v\nexpected\n%#v", i, clientMaterial, m.exported)
		}

		// Test server
		err = tc.config.ExtractSessionKeysFromDTLS(m, false)
		if err != nil {
			t.Errorf("failed to extract keys for %d-server: %v", i, err)
		}

		keys = tc.config.Keys
		serverMaterial := append([]byte{}, keys.RemoteMasterKey...)
		serverMaterial = append(serverMaterial, keys.LocalMasterKey...)
		serverMaterial = append(serverMaterial, keys.RemoteMasterSalt...)
		serverMaterial = append(serverMaterial, keys.LocalMasterSalt...)

		if !bytes.Equal(serverMaterial, m.exported) {
			t.Errorf("material reconstruction failed for %d-server:\n%#v\nexpected\n%#v", i, serverMaterial, m.exported)
		}

	}
}
