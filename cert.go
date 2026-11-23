package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// certValidity is long on purpose: an app's designated requirement names this
// exact certificate (`certificate leaf`), so replacing it makes macOS forget
// every permission users granted. codesign refuses an expired certificate, but
// apps signed before the expiry keep running.
// ponytail: one leaf for everything; a root + per-app leaves (`certificate root`)
// would allow rotation, add it once that requirement is verified against TCC.
const certValidity = 20 * 365 * 24 * time.Hour

// certInit creates a self-signed code signing identity and writes it as a
// PKCS#12 file, with its random password in `<out>.password`. Both are
// written 0600 and an existing file is never overwritten: losing or replacing
// this identity breaks the updates of every app signed with it.
func certInit(name, out string) error {
	passFile := out + ".password"
	for _, f := range []string{out, passFile} {
		if _, err := os.Stat(f); err == nil {
			return fmt.Errorf("%s already exists: refusing to replace a signing identity", f)
		}
	}

	key, certDER, err := newIdentity(name)
	if err != nil {
		return err
	}

	k, err := newKeychain()
	if err != nil {
		return err
	}
	defer k.close()

	// security has no command to build a PKCS#12 from PEM files, and the Go
	// standard library has no PKCS#12 encoder. Going through a keychain lets
	// security write the file in the format it reads back, which also avoids
	// the OpenSSL 3 default (AES) that older macOS versions cannot import.
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	for file, data := range map[string][]byte{"key.pem": keyPEM, "cert.pem": certPEM} {
		path := filepath.Join(k.dir, file)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			return err
		}
		// -A: the keychain is deleted at the end of this function, and export
		// would otherwise ask the user for permission in a dialog.
		if _, err := run("security", "import", path, "-k", k.path, "-A"); err != nil {
			return err
		}
	}

	password := randomHex(16)
	if _, err := run("security", "export", "-k", k.path, "-t", "identities", "-f", "pkcs12", "-P", password, "-o", out); err != nil {
		return err
	}
	if err := os.Chmod(out, 0o600); err != nil {
		return err
	}
	return os.WriteFile(passFile, []byte(password), 0o600)
}

// newIdentity returns a PKCS#8 RSA key and a matching self-signed certificate
// restricted to code signing, both DER-encoded.
func newIdentity(name string) (key, cert []byte, err error) {
	priv, err := rsa.GenerateKey(rand.Reader, 3072)
	if err != nil {
		return nil, nil, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 127))
	if err != nil {
		return nil, nil, err
	}
	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: name},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(certValidity),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		BasicConstraintsValid: true,
	}
	cert, err = x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, err
	}
	key, err = x509.MarshalPKCS8PrivateKey(priv)
	return key, cert, err
}
