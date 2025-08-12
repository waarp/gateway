package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

func generateCertificates(key, cert string) error {
	logger := getLogger()

	priv, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		return fmt.Errorf("cannot generate an ECDSA key: %w", err)
	}

	template := makeCertificateTemplate()

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	certFile, err := os.Create(filepath.Clean(cert))
	if err != nil {
		return fmt.Errorf("cannot open certificate file %q: %w", cert, err)
	}

	defer func() {
		if err2 := certFile.Close(); err2 != nil {
			logger.Warningf("The following error occurred whie closing the certificate file: %v", err2)
		}
	}()

	keyFile, err := os.Create(filepath.Clean(key))
	if err != nil {
		return fmt.Errorf("cannot open key file %q: %w", key, err)
	}

	defer func() {
		if err2 := keyFile.Close(); err2 != nil {
			logger.Warningf("The following error occurred while closing the key file: %v", err2)
		}
	}()

	err = pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if err != nil {
		return fmt.Errorf("cannot encode certificate to file %q: %w", cert, err)
	}

	keyBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return fmt.Errorf("cannot marshal ECDSA private key: %w", err)
	}

	err = pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})
	if err != nil {
		return fmt.Errorf("cannot encode key to file %q: %w", key, err)
	}

	return nil
}

func makeCertificateTemplate() x509.Certificate {
	const defaultCertValidity = time.Hour * 24 * 365

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"self signed certificate"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(defaultCertValidity),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	return template
}

func certificatesExist(key, cert string) bool {
	logger := getLogger()

	if !pathExists(key) {
		logger.Infof("The TLS key %q for the administration interface does not exist",
			key)

		return false
	}

	if !pathExists(cert) {
		logger.Infof("The TLS certificate %q for the administration interface does not exist",
			cert)

		return false
	}

	return true
}

func pathExists(p string) bool {
	_, err := os.Stat(p)

	return !os.IsNotExist(err)
}

// func MakeSSHKeyPair() (privKey, pubKey string, err error) {
// 	const keySize = 2048
//
// 	privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
// 	if err != nil {
// 		return "", "", fmt.Errorf("cannot generate ssh private key: %w", err)
// 	}
//
// 	privateKeyPEM := &pem.Block{
// 		Type:  "RSA PRIVATE KEY",
// 		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
// 	}
//
// 	privKeyBytes := pem.EncodeToMemory(privateKeyPEM)
//
// 	// generate and write public key
// 	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
// 	if err != nil {
// 		return "", "", fmt.Errorf("cannot marshal the ssh private key: %w", err)
// 	}
//
// 	pubKeyBytes := ssh.MarshalAuthorizedKey(pub)
//
// 	return string(privKeyBytes), string(pubKeyBytes), nil
// }

// func handleSSHKeys(partner *gwPartner) error {
// 	logger := getLogger(loggerName)
//
// 	var err error
//
// 	if partner.SSHPrivateKeyPath != "" && partner.SSHPublicKeyPath != "" {
// 		if !pathExists(partner.SSHPublicKeyPath) || !pathExists(partner.SSHPrivateKeyPath) {
// 			logger.Warning("At least one of the given public or private keys does not exist. We will try to generate them")
//
// 			if err2 := WriteNewKeyPair(partner); err2 != nil {
// 				return err2
// 			}
//
// 			return nil
// 		}
//
// 		logger.Info("Loading the SSH keypair from %q and %q",
// 			partner.SSHPrivateKeyPath, partner.SSHPublicKey)
//
// 		pubkeyBytes, err2 := ioutil.ReadFile(partner.SSHPublicKeyPath)
// 		if err2 != nil {
// 			return fmt.Errorf("cannot read the SSH public key file %q: %w",
// 				partner.SSHPublicKeyPath, err2)
// 		}
//
// 		privkeyBytes, err2 := ioutil.ReadFile(partner.SSHPrivateKeyPath)
// 		if err2 != nil {
// 			return fmt.Errorf("cannot read the SSH private key file %q: %w",
// 				partner.SSHPrivateKeyPath, err2)
// 		}
//
// 		partner.SSHPrivateKey = string(privkeyBytes)
// 		partner.SSHPublicKey = string(pubkeyBytes)
//
// 		return nil
// 	}
//
// 	partner.SSHPrivateKey, partner.SSHPublicKey, err = MakeSSHKeyPair()
// 	if err != nil {
// 		return fmt.Errorf("cannot generate an SSH keypair: %w", err)
// 	}
//
// 	return nil
// }

// func WriteNewKeyPair(partner *gwPartner) error {
// 	sshPrivateKey, sshPublicKey, err := MakeSSHKeyPair()
// 	if err != nil {
// 		return fmt.Errorf("cannot generate an SSH keypair: %w", err)
// 	}
//
// 	partner.SSHPrivateKey, partner.SSHPublicKey = sshPrivateKey, sshPublicKey
//
// 	if err = ioutil.WriteFile(partner.SSHPrivateKeyPath, []byte(partner.SSHPrivateKey), 0o600); err != nil {
// 		return fmt.Errorf("cannot write private key to %q: %w", partner.SSHPrivateKeyPath, err)
// 	}
//
// 	if err = ioutil.WriteFile(partner.SSHPublicKeyPath, []byte(partner.SSHPublicKey), 0o600); err != nil {
// 		return fmt.Errorf("cannot write public key to %q: %w", partner.SSHPublicKeyPath, err)
// 	}
//
// 	return nil
// }
