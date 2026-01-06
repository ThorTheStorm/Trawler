package crl

import (
	"crypto/x509"
	"time"
)

func validateCRLToCertificate(crlData x509.RevocationList, certData x509.Certificate) (bool, error) {
	// Verify the CRL signature using the issuer's public key
	err := crlData.CheckSignatureFrom(&certData)
	if err != nil {
		return false, err
	}
	return true, nil
}

func CheckIfCRLIsValid(crlData x509.RevocationList, certData x509.Certificate) (bool, error) {
	// validate timestamps, signatures, etc.

	// Check if the CRL is expired
	timeNow := time.Now()
	if crlData.NextUpdate.Before(timeNow) {
		return false, nil
	}

	valid, err := validateCRLToCertificate(crlData, certData)
	if err != nil {
		return valid, err
	}

	return valid, nil
}
